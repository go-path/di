package di

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

type ctxCurrentInCreationKeyType int // unexported type for ctxCurrentInCreationKey to avoid collisions.

type Container interface {

	// Initialize initialize all non-lazy singletons (startup)
	Initialize(ctx ...context.Context) error

	// RegisterScope Register the given scope, backed by the given ScopeI implementation.
	RegisterScope(name string, scope ScopeI) error

	Register(ctor any, opts ...FactoryConfig)

	ShouldRegister(ctor any, opts ...FactoryConfig) error

	// Get return an instance, which may be shared or independent, of the specified component.
	Get(key reflect.Type, ctx ...context.Context) (any, error)

	// Contains check if this container contains a component with the given key.
	// Does not consider any hierarchy this container may participate in.
	Contains(key reflect.Type) bool

	Filter(options ...FactoryConfig) *FilteredFactories

	GetObjectFactory(factory *Factory, managed bool, ctx ...context.Context) ObjectFactory

	GetObjectFactoryFor(key reflect.Type, managed bool, ctx ...context.Context) ObjectFactory

	// ResolveArgs returns an ordered list of values which may be passed directly to the Factory Create method
	ResolveArgs(factory *Factory, ctx ...context.Context) ([]reflect.Value, error)

	// Destroy this container
	Destroy() error

	// DestroyObject destroy the given instance
	DestroyObject(key reflect.Type, object any) error

	// DestroySingletons destroy all singleton components in this container. To be called on shutdown of a factory.
	DestroySingletons() error

	// Mock test only, register a mock instance to the container
	Mock(mock any) (cleanup func())
}

type container struct {
	locked         bool // by design, we lock the container after initialization
	graph          *graph
	parent         Container
	paramsMu       sync.RWMutex
	mockMu         sync.Mutex
	scopes         map[string]ScopeI
	knownParams    map[reflect.Type]*Parameter
	factories      map[reflect.Type][]*Factory
	singletons     *scopeSingleton
	testingHasMock bool
	testingMocks   map[reflect.Type]mockFunc
}

var (
	ctxCurrentInCreationKey ctxCurrentInCreationKeyType
	initializersStereotype  = Stereotype(Singleton, Condition(func(c Container, f *Factory) bool {
		return f.Startup()
	}))
	ErrCycleDetected         = errors.New("this component introduces a cycle")
	ErrManyCandidates        = errors.New("multiple candidates found")
	ErrContainerLocked       = errors.New("container is locked")
	ErrContextRequired       = errors.New("context required")
	ErrInvalidProvider       = errors.New("invalid provider")
	ErrMissingDependency     = errors.New("missing dependencies")
	ErrCandidateNotFound     = errors.New("no candidate found")
	ErrNoScopeNameDefined    = errors.New("no scope name defined for component")
	ErrCurrentlyInCreation   = errors.New("requested bean is currently in creation")
	ErrNoScopeNameRegistered = errors.New("no Scope registered")
)

func New(parent Container) Container {
	c := &container{
		graph:          &graph{},
		parent:         parent,
		scopes:         make(map[string]ScopeI),
		factories:      make(map[reflect.Type][]*Factory),
		singletons:     newSingletonScope(),
		testingHasMock: false,
		testingMocks:   make(map[reflect.Type]mockFunc),
		knownParams:    make(map[reflect.Type]*Parameter),
	}

	c.scopes[SCOPE_SINGLETON] = c.singletons
	c.scopes[SCOPE_PROTOTYPE] = &scopePrototypeImpl{}

	c.graph.container = c
	return c
}

func (c *container) Initialize(contexts ...context.Context) error {
	c.paramsMu.Lock()
	if c.locked {
		c.paramsMu.Unlock()
		return ErrContainerLocked
	}
	c.locked = true

	ctx := getContext(contexts...)

	// update candidates alias
	c.refreshAliasAll()

	c.paramsMu.Unlock()

	return c.Filter(initializersStereotype).Foreach(func(f *Factory) (bool, error) {
		if _, _, err := c.GetObjectFactory(f, true, ctx)(); err != nil {
			return true, err
		}

		return false, nil
	})
}

func (c *container) RegisterScope(name string, scope ScopeI) error {
	if c.locked {
		return ErrContainerLocked
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("scope identifier must not be empty")
	}

	if SCOPE_SINGLETON == name || SCOPE_PROTOTYPE == name {
		return errors.New("cannot replace existing scopes 'singleton' and 'prototype'")
	}

	c.scopes[name] = scope

	return nil
}

func (c *container) Register(ctor any, opts ...FactoryConfig) {
	if err := c.ShouldRegister(ctor, opts...); err != nil {
		panic(err)
	}
}

func (c *container) ShouldRegister(funcOrRef any, options ...FactoryConfig) error {
	if c.locked {
		return ErrContainerLocked
	}
	factoryType := reflect.TypeOf(funcOrRef)
	if factoryType == nil {
		return errors.Join(errors.New("can't register an untyped nil"), ErrInvalidProvider)
	}

	var paramsKeys []reflect.Type
	var returnType reflect.Type

	returnErrorIdx := -1
	returnValueIdx := -1

	isSingletonInstance := false

	if factoryType.Kind() != reflect.Func {
		isSingletonInstance = true
		returnType = factoryType
		instanceRef := funcOrRef
		funcOrRef = func() any {
			return instanceRef
		}
		returnValueIdx = 0
		factoryType = reflect.TypeOf(funcOrRef)
	} else {
		// builds a params list from the provided constructor type.
		numParams := factoryType.NumIn()
		if factoryType.IsVariadic() {
			numParams--
		}

		paramsKeys = make([]reflect.Type, 0, numParams)
		for i := 0; i < numParams; i++ {
			paramsKeys = append(paramsKeys, KeyOf(factoryType.In(i)))
		}

		// results
		// func([context.Context], [dep A..N]) ([ServiceA], [error])
		numOut := factoryType.NumOut()
		if numOut == 0 {
			// used by initializers (Startup)
			returnType = _typeNilReturn
		} else {
			returnType = factoryType.Out(0)
			switch numOut {
			case 1:
				if isError(returnType) {
					// ex. "func() (error)"
					returnErrorIdx = 0
					returnType = _typeNilReturn
				} else {
					// ex. "func() (Service)"
					returnValueIdx = 0
				}
			case 2:
				returnTypeB := factoryType.Out(1)

				if isError(returnType) {
					// ex. "func() (error, Service)"

					if isError(returnTypeB) {
						// ex. "func() (error, error)"
						return errors.Join(fmt.Errorf("%v has invalid returns", factoryType), ErrInvalidProvider)
					}
					returnType = returnTypeB
					returnErrorIdx = 0
					returnValueIdx = 1
				} else {
					// ex. "func() (Service, error)"

					if !isError(returnTypeB) {
						// ex. "func() (ServiceA, ServiceA)"
						return errors.Join(fmt.Errorf("%v has invalid returns", factoryType), ErrInvalidProvider)
					}

					returnErrorIdx = 1
					returnValueIdx = 0
				}
			default:
				return errors.Join(fmt.Errorf("%v has invalid return", factoryType), ErrInvalidProvider)
			}
		}
	}

	returnKey := KeyOf(returnType)
	factoryValue := reflect.ValueOf(funcOrRef)

	if returnType != _typeNilReturn && returnErrorIdx != -1 {
		if returnErrorIdx == 0 {
			returnValueIdx = 1
		}
	}

	factory := &Factory{
		key:            returnKey,
		factoryType:    factoryType,
		factoryValue:   factoryValue,
		returnType:     returnType,
		returnErrorIdx: returnErrorIdx,
		returnValueIdx: returnValueIdx,
		parameterKeys:  paramsKeys,
		scope:          SCOPE_SINGLETON,
		qualifiers:     make(map[reflect.Type]bool),
	}

	if factory.ReturnsValue() {
		if returnType.Implements(_typeInitializer) {
			initializer := func(object any) {
				if d, ok := object.(Initializable); ok {
					d.Initialize()
				}
			}
			factory.initializers = append(factory.initializers, reflect.ValueOf(initializer))
		}

		if returnType.Implements(_typeDisposable) {
			disposer := func(object any) {
				if d, ok := object.(Disposable); ok {
					d.Destroy()
				}
			}
			factory.disposers = append(factory.disposers, disposer)
		}
	}

	for _, option := range options {
		option(factory)
	}

	if isSingletonInstance {
		factory.scope = SCOPE_SINGLETON
	}

	// ignore if the factory returns nil
	if !factory.ReturnsValue() {
		factory.disposers = nil
		factory.initializers = nil
	}

	// a component is only eligible for registration when all specified conditions match
	for _, match := range factory.conditions {
		if !match(c, factory) {
			return nil
		}
	}

	// cache old providers before running cycle detection.
	oldFactories := c.factories[returnKey]
	c.factories[returnKey] = append(c.factories[returnKey], factory)

	factory.order = c.graph.add(factory)
	if ok, cycle := c.graph.isAcyclic(); !ok {
		// When a cycle is detected, recover the old providers to reset
		// the providers map back to what it was before this node was
		// introduced.
		c.factories[returnKey] = oldFactories
		fmt.Println(cycle)
		return ErrCycleDetected
	}

	// update cache
	c.GetParam(returnKey)
	for _, paramKey := range paramsKeys {
		factory.parameters = append(factory.parameters, c.GetParam(paramKey))
	}

	return nil
}

// GetParam get param information
func (c *container) GetParam(key reflect.Type) *Parameter {
	c.paramsMu.RLock()
	param, exists := c.knownParams[key]
	c.paramsMu.RUnlock()
	if !exists {
		c.paramsMu.Lock()
		param, exists = c.knownParams[key] // 2nd check
		if !exists {
			param = c.parseParam(key)
			c.knownParams[key] = param
			if c.locked {
				// update candidate list (alias)
				c.refreshAlias(param)
			}
		}
		c.paramsMu.Unlock()

		if param.Qualified() || param.Provider() {
			// cache value type too
			c.GetParam(param.Value())
		}
	}
	return param
}

// parseParam extract param information from a type
func (c *container) parseParam(paramKey reflect.Type) *Parameter {

	var isProvider bool
	var isUnmanaged bool
	var isQualified bool
	valueType := paramKey

	var qualifierType reflect.Type
	var funcWithImpl func(any) reflect.Value

	if paramKey.Kind() == reflect.Struct {

		isProvider = strings.HasPrefix(paramKey.String(), "di.Provider[")
		isUnmanaged = strings.HasPrefix(paramKey.String(), "di.Unmanaged[")
		isQualified = strings.HasPrefix(paramKey.String(), "di.Qualified[")

		if isUnmanaged {
			isProvider = true
		}

		if isProvider || isQualified {
			var funcWith reflect.Method
			var funcType reflect.Method
			var funcQualifier reflect.Method

			// func (q Provider[T]) With(supplier func() (any, error)) Provider[T]
			// func (q Qualified[T, Q]) With(value any) Qualified[T, Q]
			funcWith, isProvider = paramKey.MethodByName("With")
			if !isProvider || funcWith.Type.NumIn() != 2 || funcWith.Type.NumOut() != 1 {
				isProvider = false
			}

			if isProvider || isQualified {
				funcType, isQualified = paramKey.MethodByName("Type")
				if !isQualified || funcType.Type.NumIn() != 1 || funcType.Type.NumOut() != 1 || !funcType.Type.Out(0).Implements(_typeReflectType) {
					isQualified = false
				}
			}

			// Value {di.Provider[Service]}
			// Value {di.Qualified[Service, Qualifier]}
			nptr_vl := reflect.New(paramKey).Elem()

			if isProvider && isQualified {
				funcQualifier, isQualified = paramKey.MethodByName("Qualifier")
				if !isQualified || funcQualifier.Type.NumIn() != 1 || funcQualifier.Type.NumOut() != 1 || !funcQualifier.Type.Out(0).Implements(_typeReflectType) {
					isQualified = false
				}

				if isQualified {
					isProvider = false
					qualifierResult := funcQualifier.Func.Call([]reflect.Value{nptr_vl})
					qualifierType = qualifierResult[0].Interface().(reflect.Type)
				}
			}

			if isProvider || isQualified {
				valueTypeResult := funcType.Func.Call([]reflect.Value{nptr_vl})
				valueType = valueTypeResult[0].Interface().(reflect.Type)

				funcWithImpl = func(value any) reflect.Value {
					// func (q Qualified[T, Q]) With(value any) Qualified[T, Q]
					// func (p Provider[T]) With(supplier func() (any, error)) Provider[T]
					// func (u Unmanaged[T]) With(supplier func() (any, DisposableAdapter, error)) Unmanaged[T]
					result := funcWith.Func.Call([]reflect.Value{nptr_vl, reflect.ValueOf(value)})
					return result[0]
				}
			}

			if isUnmanaged {
				var funcUnmanaged reflect.Method
				funcUnmanaged, isUnmanaged = paramKey.MethodByName("Unmanaged")
				if !isUnmanaged || funcUnmanaged.Type.NumIn() != 1 || funcUnmanaged.Type.NumOut() != 1 {
					isUnmanaged = false
				}
			}
		}
	}

	return &Parameter{
		key:          paramKey,
		value:        valueType,
		provider:     isProvider,
		unmanaged:    isUnmanaged,
		qualified:    isQualified,
		qualifier:    qualifierType,
		funcWithImpl: funcWithImpl,
		factories:    make(map[*Factory]bool),
		candidates:   make(map[*Factory]bool),
	}
}

func (c *container) refreshAliasAll() {
	c.refreshAliasFn(func(f func(*Parameter)) {
		for _, param := range c.knownParams {
			f(param)
		}
	})
}

func (c *container) refreshAlias(params ...*Parameter) {
	c.refreshAliasFn(func(f func(*Parameter)) {
		for _, param := range params {
			f(param)
		}
	})
}

func (c *container) refreshAliasFn(loop func(func(*Parameter))) {
	for returnType, factories := range c.factories {
		if returnType == _typeNilReturn {
			continue
		}
		loop(func(p *Parameter) {
			paramType := p.Key()
			if paramType == _typeNilReturn {
				return
			}

			for _, f := range factories {
				if _, exist := p.factories[f]; exist {
					continue
				}
				if _, exist := p.candidates[f]; exist {
					continue
				}

				isCandidate, isExactMatch := p.IsValidCandidate(f)
				if !isCandidate {
					continue
				}

				if isExactMatch {
					p.factories[f] = true
				} else {
					p.candidates[f] = true
					slog.Info(fmt.Sprintf("[di] '%s' is a candidate for '%s'\n", returnType.String(), paramType.String()))
				}
			}
		})
	}
}

// Get a managed component (by scope)
func (c *container) Get(key reflect.Type, contexts ...context.Context) (instance any, e error) {
	ctx := getContext(contexts...)

	param := c.GetParam(key)

	// eagerly check singleton cache for manually registered singletons.
	if singleton := c.singletons.getSingleton(key); singleton != nil {
		instance = singleton
		return
	}

	// fail if we're already creating this instance: we're assumably within a circular reference.
	if c.isInCreation(key, ctx) {
		e = ErrCurrentlyInCreation
		return
	}

	// Check if component exists in this container
	if c.parent != nil && !c.Contains(key) {
		// not found -> check parent.
		return c.parent.Get(key, ctx)
	}

	var factory *Factory
	factory, e = c.getProvider(param)
	if e != nil {
		return
	}

	instance, _, e = c.createObject(factory, ctx, true)
	return
}

// ObjectFactory get a factory for a managed component (by scope)
func (c *container) GetObjectFactory(factory *Factory, managed bool, contexts ...context.Context) ObjectFactory {
	return func() (any, DisposableAdapter, error) {
		return c.createObject(factory, getContext(contexts...), managed)
	}
}

func (c *container) GetObjectFactoryFor(key reflect.Type, managed bool, contexts ...context.Context) ObjectFactory {

	// Check if component exists in this container
	if c.parent != nil && !c.Contains(key) {
		// not found -> check parent.
		return c.parent.GetObjectFactoryFor(key, managed, contexts...)
	}

	param := c.GetParam(key)
	var factory *Factory
	factory, e := c.getProvider(param)

	return func() (any, DisposableAdapter, error) {
		if e != nil {
			return nil, nil, e
		}

		return c.createObject(factory, getContext(contexts...), managed)
	}
}

func (c *container) createObject(factory *Factory, ctx context.Context, managed bool) (instance any, disposer DisposableAdapter, e error) {
	if factory.mock != nil {
		instance = factory.mock(ctx)
		return
	}

	if err := c.checkMissingDependencies(factory); err != nil {
		e = errors.Join(ErrMissingDependency, fmt.Errorf("%v depends on missing dependency", factory.factoryType), err)
		return
	}

	// args
	var args []reflect.Value
	if args, e = c.ResolveArgs(factory, ctx); e != nil {
		return
	}

	key := factory.key

	objectFactory := func() (out any, disposer DisposableAdapter, err error) {
		defer func() {
			if err == nil && factory.ReturnsValue() && len(factory.initializers) > 0 {
				// instance created - initializers/post construct
				args := []reflect.Value{reflect.ValueOf(out)}
				for _, callback := range factory.initializers {
					callback.Call(args)
				}
			}
			ctx = c.afterCreation(key, ctx)
		}()
		ctx = c.beforeCreation(key, ctx)

		if out, err = factory.Create(args); err != nil {
			return
		} else if factory.HasDisposers() {
			// add the instance to the list of disposable components in this container
			disposer = &disposableAdapterImpl{
				obj:       out,
				factory:   factory,
				container: c,
				getContext: func() context.Context {
					return ctx
				},
			}
		}

		return
	}

	if managed {
		scopeName := factory.scope

		if scopeName == "" {
			e = errors.Join(fmt.Errorf("no scope name defined for component %v", factory), ErrNoScopeNameDefined)
			return
		}

		scope := c.scopes[scopeName]
		if scope == nil {
			e = errors.Join(fmt.Errorf("no scope registered for name %s", scopeName), ErrNoScopeNameRegistered)
			return
		}

		// create component instance
		instance, e = scope.Get(ctx, key, objectFactory)
	} else {
		instance, disposer, e = objectFactory()
	}

	return
}

// ResolveArgs returns an ordered list of values which may be passed directly to the Factory Create method
func (c *container) ResolveArgs(factory *Factory, contexts ...context.Context) ([]reflect.Value, error) {
	ctx := getContext(contexts...)
	args := make([]reflect.Value, len(factory.parameters))
	for i, param := range factory.parameters {
		paramKey := param.Key()
		if paramKey == _keyContext {
			args[i] = reflect.ValueOf(ctx)
		} else if paramKey == _keyContainer {
			args[i] = reflect.ValueOf(c)
		} else if param.Provider() {

			// async get
			objectFactory := c.GetObjectFactoryFor(param.Value(), !param.Unmanaged())
			if param.Unmanaged() {
				// user will be responsible for cleaning them up (call disposable.Dispose())
				args[i] = param.ValueOf(func() (any, DisposableAdapter, error) {
					object, disposable, err := objectFactory()
					return object, disposable, err
				})
			} else {
				// managed by scope (Ex. Request Scoped will destroy any Scoped("request"))
				args[i] = param.ValueOf(func() (any, error) {
					object, _, err := objectFactory()
					return object, err
				})
			}

		} else {
			isQualifier := param.Qualified()

			if isQualifier {
				paramKey = param.Value()
			}

			value, err := c.Get(paramKey, ctx)
			if err != nil {
				return nil, err
			}

			if isQualifier {
				args[i] = param.ValueOf(value)
			} else {
				args[i] = reflect.ValueOf(value)
			}
		}
	}
	return args, nil
}

// Checks that all direct dependencies of the provided parameters are present in
// the container. Returns an error if not.
func (c *container) checkMissingDependencies(f *Factory) error {

	var missingDeps []string

	for _, param := range f.parameters {
		paramKey := param.Key()
		if paramKey == _keyContext || paramKey == _keyContainer {
			// ignore context.Context and Container
			continue
		}

		// allProviders := c.factories[paramKey]
		// This means that there is no factory that provides this value,
		// and it is NOT being decorated and is NOT optional.
		// In the case that there is no providers but there is a decorated value
		// of this type, it can be provided safely so we can safely skip this.
		if !param.HasCandidates() {
			if c.testingHasMock { // see Mock
				if _, ok := c.testingMocks[paramKey]; ok {
					continue
				}
			}

			missingDeps = append(missingDeps, fmt.Sprintf("%v", paramKey))
		}
	}

	if len(missingDeps) == 0 {
		return nil
	}

	return errors.New("missing dependencies: " + strings.Join(missingDeps, ", "))
}

// isInCreation return whether the specified key is currently in creation.
func (c *container) isInCreation(key reflect.Type, ctx context.Context) bool {
	if curVal := ctx.Value(ctxCurrentInCreationKey); curVal != nil {
		if curVal == key {
			return true
		} else if keyMap, isType := curVal.(map[reflect.Type]bool); isType {
			return keyMap[key]
		}
	}
	return false
}

// beforeCreation callback before object creation. Registers the key as currently in creation.
func (c *container) beforeCreation(key reflect.Type, ctx context.Context) context.Context {
	curVal := ctx.Value(ctxCurrentInCreationKey)
	if curVal == nil {
		keyMap := map[reflect.Type]bool{}
		keyMap[key] = true
		return context.WithValue(ctx, ctxCurrentInCreationKey, keyMap)
	} else {
		curVal.(map[reflect.Type]bool)[key] = true
		return ctx
	}
}

// afterCreation callback after object creation. Marks the key as not in creation anymore.
func (c *container) afterCreation(key reflect.Type, ctx context.Context) context.Context {
	curVal := ctx.Value(ctxCurrentInCreationKey)
	if keyMap, isType := curVal.(map[reflect.Type]bool); isType {
		delete(keyMap, key)
		if len(keyMap) == 0 {
			return context.WithValue(ctx, ctxCurrentInCreationKey, nil)
		}
	}
	return ctx
}

func (c *container) Contains(key reflect.Type) bool {
	if c.testingHasMock {
		if _, ok := c.testingMocks[key]; ok {
			return true
		}
	}

	param := c.GetParam(key)
	return param.HasCandidates()
}

func (c *container) getProvider(p *Parameter) (*Factory, error) {

	key := p.Key()

	// see Mock
	if c.testingHasMock {
		if fn, ok := c.testingMocks[key]; ok {
			return &Factory{mock: fn}, nil
		}
	}

	// Get candidates
	var candidates []*Factory

	if len(p.factories) > 0 {
		for f := range p.factories {
			candidates = append(candidates, f)
		}
	} else if len(p.candidates) > 0 {
		// has compatible alias
		for f := range p.candidates {
			candidates = append(candidates, f)
		}
	}

	switch len(candidates) {
	case 0:
		return nil, errors.Join(fmt.Errorf("no candidate found for type %v", key), ErrCandidateNotFound)
	case 1:
		return candidates[0], nil
	default:
		sort.Slice(candidates, func(i, j int) bool {
			return DefaultFactorySortLessFn(candidates[i], candidates[j])
		})

		first := candidates[0]
		second := candidates[1]

		if first.Mock() {
			// testing, expected controlled environment
			return first, nil
		}

		if first.Primary() {
			if first.Priority() > second.Priority() || !second.Primary() {
				// If exactly one 'primary' component exists among the candidates, it
				// will be the injected value.
				return first, nil
			}
		} else if !first.Alternative() && second.Alternative() {
			// If exactly one NON-ALTERNATIVE component exists among the candidates, it
			// will be the injected value.
			return first, nil
		}

		if first.Priority() > second.Priority() {
			// The candidate with the highest priority will be injected.
			return first, nil
		}

		return nil, errors.Join(fmt.Errorf("multiple candidates for type %v", key), ErrManyCandidates)
	}
}

func (c *container) Destroy() error {
	// c.Cleanup()
	// c.graph = &graph{container: c}
	// c.factories = make(map[reflect.Type][]*Factory)
	// c.singletonStorage = &storage{}
	// c.testingHasMock = false
	// c.testingMocks = make(map[reflect.Type]mockFn)
	return nil
}

func (c *container) DestroyObject(key reflect.Type, object any) error {
	return nil
}

func (c *container) DestroySingletons() error {
	return nil
}

// Mock allows mocking of a dependency. Accepts "any", "func() any" or "func(context.Context) any"
func (c *container) Mock(mock any) (cleanup func()) {
	if !testing.Testing() {
		panic("mocks are only allowed during testing")
	}

	var fn mockFunc
	var key reflect.Type

	if tFunc, isFunc := mock.(func() any); isFunc {
		key = reflect.PointerTo(reflect.TypeOf(mock).Out(0))
		fn = func(ctx context.Context) any {
			return tFunc()
		}
	} else if tFuncCtx, isFuncCtx := mock.(func(ctx context.Context) any); isFuncCtx {
		key = reflect.PointerTo(reflect.TypeOf(mock).Out(0))
		fn = func(ctx context.Context) any {
			return tFuncCtx(ctx)
		}
	} else {
		key = reflect.PointerTo(reflect.TypeOf(mock))
		fn = func(ctx context.Context) any {
			return mock
		}
	}

	c.mockMu.Lock()
	defer c.mockMu.Unlock()

	c.testingHasMock = true
	c.testingMocks[key] = fn

	// cleanup
	cleanup = func() {
		c.mockMu.Lock()
		defer c.mockMu.Unlock()

		delete(c.testingMocks, key)
		c.testingHasMock = len(c.testingMocks) > 0
	}
	return
}
