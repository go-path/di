package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type ctxCurrentInCreationKeyType int // unexported type for ctxCurrentInCreationKey to avoid collisions.

type Container interface {

	// Initialize initialize all non-lazy singletons (startup)
	Initialize(contexts ...context.Context) error

	// RegisterScope Register the given scope, backed by the given ScopeI implementation.
	RegisterScope(name string, scope ScopeI) error

	Register(ctor any, opts ...ComponentConfig)

	ShouldRegister(ctor any, opts ...ComponentConfig) error

	// Get return an instance, which may be shared or independent, of the specified component.
	Get(key reflect.Type, contexts ...context.Context) (any, error)

	// Contains check if this container contains a component with the given key.
	// Does not consider any hierarchy this container may participate in.
	Contains(key reflect.Type) bool

	Filter(options ...ComponentConfig) *FilteredFactories

	ObjectFactory(factory *Factory, contexts ...context.Context) ObjectFactory

	// ResolveArgs returns an ordered list of values which may be passed directly to the Factory Create method
	ResolveArgs(ctx context.Context, parameterTypes []reflect.Type) ([]reflect.Value, error)

	// Destroy this container
	Destroy() error

	// DestroyObject destroy the given instance
	DestroyObject(key reflect.Type, object any) error

	// DestroySingletons destroy all singleton components in this container. To be called on shutdown of a factory.
	DestroySingletons() error

	// Mock test only, register a mock instance to the container
	Mock(mock any) (cleanup func())

	// TypeMatch check whether the component with the given type matches the specified type.
	// More specifically, check whether a Get call for the given type would return an object
	// that is assignable to the specified target type
	// TypeMatch(key reflect.Type, typeToMatch Resolvable) (bool, error)
	// GetFactory(key Resolvable, contexts ...context.Context) (*Factory, error)
	// RunOnStartup(ctor any, priority int, opts ...ComponentOption)
	// Instances(ctx context.Context, valid func(p *Factory) bool, less func(a, b *Factory) bool) (instances []any, err error)
	// Foreach(visitor func(p *Factory) (stop bool, err error)) error
	// AddContext(ctx context.Context) context.Context
}

type container struct {
	locked         bool // by design, we lock the container after initialization
	graph          *graph
	parent         Container
	mockMu         sync.Mutex
	allKnownKeys   map[reflect.Type]bool                  // dependencias conhecidas
	alias          map[reflect.Type]map[reflect.Type]bool // tipos compatíveis
	scopes         map[string]ScopeI
	factories      map[reflect.Type][]*Factory
	singletons     *scopeSingleton
	testingHasMock bool
	testingMocks   map[reflect.Type]mockFn
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
		alias:          make(map[reflect.Type]map[reflect.Type]bool), // {A:{B:true,C:true}}, if A is missing we can use B instead
		scopes:         make(map[string]ScopeI),
		factories:      make(map[reflect.Type][]*Factory),
		singletons:     newSingletonScope(),
		testingHasMock: false,
		testingMocks:   make(map[reflect.Type]mockFn),
		allKnownKeys:   make(map[reflect.Type]bool),
	}

	c.scopes[SCOPE_SINGLETON] = c.singletons
	c.scopes[SCOPE_PROTOTYPE] = &scopePrototypeImpl{}

	c.graph.container = c
	return c
}

func (c *container) Initialize(contexts ...context.Context) error {
	if c.locked {
		return ErrContainerLocked
	}
	c.locked = true

	ctx := getContext(contexts...)

	// update candidates alias
	for key := range c.factories {
		returnType := key.Elem()
		if returnType == _typeNilReturn {
			continue
		}
		for paramKey := range c.allKnownKeys {
			paramType := paramKey.Elem()

			if returnType == paramType {
				continue
			}

			isCompatible := returnType.AssignableTo(paramType)

			if isCompatible {
				mm, exist := c.alias[paramKey]
				if !exist {
					mm = make(map[reflect.Type]bool)
					c.alias[paramKey] = mm
				}
				mm[key] = true
				fmt.Printf("[di] '%v' is a candidate for '%v'\n", returnType, paramType)
			}
		}
	}

	return c.Filter(initializersStereotype).Sort(nil).Foreach(func(f *Factory) (bool, error) {
		if _, _, err := c.ObjectFactory(f, ctx)(); err != nil {
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

func (c *container) Register(ctor any, opts ...ComponentConfig) {
	if err := c.ShouldRegister(ctor, opts...); err != nil {
		panic(err)
	}
}

func (c *container) ShouldRegister(funcOrRef any, options ...ComponentConfig) error {
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
				if d, ok := object.(InitializerComponent); ok {
					d.Initialize()
				}
			}
			factory.initializers = append(factory.initializers, reflect.ValueOf(initializer))
		}

		if returnType.Implements(_typeDisposable) {
			disposer := func(object any) {
				if d, ok := object.(DisposableComponent); ok {
					d.Destroy()
				}
			}
			factory.disposers = append(factory.disposers, reflect.ValueOf(disposer))
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

	// update alias
	c.allKnownKeys[returnKey] = true
	for _, paramKey := range paramsKeys {
		if c.allKnownKeys[paramKey] {
			continue
		}
		c.allKnownKeys[paramKey] = true
	}

	return nil
}

// ParseParam extract param information from a type
func (c *container) ParseParam(key reflect.Type) *Parameter {
	return nil
}

func (c *container) Get(key reflect.Type, contexts ...context.Context) (instance any, e error) {
	ctx := getContext(contexts...)

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

	factory, e = c.getProvider(key)
	if e != nil {
		return
	}

	instance, _, e = c.createObject(factory, ctx)
	return
}

func (c *container) ObjectFactory(factory *Factory, contexts ...context.Context) ObjectFactory {
	return func() (any, DisposableAdapter, error) {
		return c.createObject(factory, getContext(contexts...))
	}
}

func (c *container) createObject(factory *Factory, ctx context.Context) (instance any, disposer DisposableAdapter, e error) {
	if factory.mock != nil {
		instance = factory.mock(ctx)
		return
	}

	key := factory.key

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

	if err := c.checkMissingDependencies(factory.parameterKeys); err != nil {
		e = errors.Join(ErrMissingDependency, fmt.Errorf("%v depends on missing dependency", factory.factoryType), err)
		return
	}

	// args
	var args []reflect.Value
	if args, e = c.ResolveArgs(ctx, factory.parameterKeys); e != nil {
		return
	}

	instanceCreated := false

	// create component instance
	instance, e = scope.Get(ctx, key, func() (any, DisposableAdapter, error) {
		defer func() {
			instanceCreated = true
			ctx = c.afterCreation(key, ctx)
		}()
		ctx = c.beforeCreation(key, ctx)

		if obj, err := factory.Create(args); err != nil {
			return nil, nil, err
		} else if factory.Disposer() {
			// add the instance to the list of disposable components in this container
			disposer = &disposableAdapterImpl{
				obj:       obj,
				factory:   factory,
				container: c,
				getContext: func() context.Context {
					return ctx
				},
			}
			return obj, disposer, nil
		} else {
			return obj, nil, nil
		}
	})

	if e == nil && instanceCreated && factory.ReturnsValue() && len(factory.initializers) > 0 {
		// initializers/post construct
		postConstructArgs := []reflect.Value{reflect.ValueOf(instance)}
		for _, callback := range factory.initializers {
			callback.Call(postConstructArgs)
		}
	}

	return
}

// ResolveArgs returns an ordered list of values which may be passed directly to the Factory Create method
func (c *container) ResolveArgs(ctx context.Context, parametersKeys []reflect.Type) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(parametersKeys))
	for i, paramKey := range parametersKeys {
		if isContext(paramKey) {
			args[i] = reflect.ValueOf(ctx)
		} else {
			// if p.useContext && i == 0 {
			// 	args[0] = reflect.ValueOf(ctx)
			// 	continue
			// }
			v, err := c.Get(paramKey, ctx)
			if err != nil {
				return nil, err
			}
			args[i] = reflect.ValueOf(v)
		}
	}
	return args, nil
}

// Checks that all direct dependencies of the provided parameters are present in
// the container. Returns an error if not.
func (c *container) checkMissingDependencies(parametersKeys []reflect.Type) error {

	var missingDeps []string

	for _, paramKey := range parametersKeys {
		if isContext(paramKey) {
			// ignore context
			continue
		}

		allProviders := c.factories[paramKey]
		// This means that there is no factory that provides this value,
		// and it is NOT being decorated and is NOT optional.
		// In the case that there is no providers but there is a decorated value
		// of this type, it can be provided safely so we can safely skip this.
		if len(allProviders) == 0 {
			if c.testingHasMock { // see Mock
				if _, ok := c.testingMocks[paramKey]; ok {
					continue
				}
			}

			missingDeps = append(missingDeps, fmt.Sprintf("%v", paramKey.Elem()))
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
	if len(c.factories[key]) > 0 {
		return true
	}

	return false

}

func (c *container) updateAliases() {

}

func (c *container) getProvider(key reflect.Type) (*Factory, error) {

	// TypeMatch check whether the component with the given type matches the specified type.
	// More specifically, check whether a Get call for the given type would return an object
	// that is assignable to the specified target type
	// TypeMatch(key reflect.Type, typeToMatch Resolvable) (bool, error)
	// GetFactory(key Resolvable, contexts ...context.Context) (*Factory, error)

	// see Mock
	if c.testingHasMock {
		if fn, ok := c.testingMocks[key]; ok {
			return &Factory{mock: fn}, nil
		}
	}

	// Get candidates
	candidates := c.factories[key]

	if len(candidates) == 0 && len(c.alias[key]) > 0 {
		// has compatible alias
		for otherKey := range c.alias[key] {
			otherCandidates := c.factories[otherKey]
			if len(otherCandidates) > 0 {
				candidates = append(candidates, otherCandidates...)
			}
		}
	}

	switch len(candidates) {
	case 0:
		return nil, errors.Join(fmt.Errorf("no candidate found for type %v", key), ErrCandidateNotFound)
	case 1:
		return candidates[0], nil
	default:
		// check for primary
		for _, candidate := range candidates {
			if candidate.primary {
				return candidate, nil
			}
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

// Mock allows mocking of a dependency. Accepts "T" or "func([context.Context]) T"
func (c *container) Mock(mock any) (cleanup func()) {
	if !testing.Testing() {
		panic("mocks are only allowed during testing")
	}

	var fn mockFn
	var key reflect.Type

	if tFn, isFn := mock.(func() any); isFn {
		key = reflect.PointerTo(reflect.TypeOf(mock).Out(0))
		fn = func(ctx context.Context) any {
			return tFn()
		}
	} else if tFnCtx, isFnCtx := mock.(func(ctx context.Context) any); isFnCtx {
		key = reflect.PointerTo(reflect.TypeOf(mock).Out(0))
		fn = func(ctx context.Context) any {
			return tFnCtx(ctx)
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

// func (c *container) TypeMatch(key reflect.Type, typeToMatch Resolvable) (bool, error) {
// 	return false, nil
// }

// func (c *container) GetFactory(key Resolvable, contexts ...context.Context) (*Factory, error) {
// 	return nil, nil
// }

// func (c *container) RunOnStartup(ctor any, priority int, opts ...ComponentConfig) {
// 	c.Register(ctor, append([]ComponentConfig{Startup(priority)}, opts...)...)
// }

// func (c *container) Instances(ctx context.Context, valid func(f *Factory) bool, less func(a, b *Factory) bool) (instances []any, err error) {

// 	var factories []*Factory

// 	for _, candidates := range c.factories {
// 		for _, p := range candidates {
// 			if valid(p) {
// 				factories = append(factories, p)
// 			}
// 		}
// 	}

// 	if less == nil {
// 		less = FactorySortLessFn
// 	}

// 	sort.Slice(factories, func(i, j int) bool {
// 		return less(factories[i], factories[j])
// 	})

// 	for _, f := range factories {
// 		if f.mock != nil {
// 			instances = append(instances, f.mock(ctx))
// 			continue
// 		}

// 		// @TODO: args
// 		if instance, e := f.Create(nil); e != nil {
// 			err = e
// 			return
// 		} else if instance != nil {
// 			// return
// 			instances = append(instances, instance)
// 		}
// 	}

// 	return
// }

// func (c *container) Foreach(visitor func(f *Factory) (stop bool, err error)) error {
// 	for _, candidates := range c.factories {
// 		for _, p := range candidates {
// 			stop, err2 := visitor(p)
// 			if err2 != nil {
// 				return err2
// 			}
// 			if stop {
// 				return nil
// 			}
// 		}
// 	}
// 	return nil
// }

// func (c *container) Cleanup() {
// 	c.contextStorages.Range(func(key, value any) bool {
// 		c.contextStorages.Delete(key)
// 		if s, ok := value.(*storage); ok {
// 			s.cleanup()
// 		}
// 		return true
// 	})
// 	c.singletonStorage.cleanup()
// }

// func (c *container) AddContext(ctx context.Context) context.Context {
// 	id := ctxSeq.Add(1)
// 	ctx = context.WithValue(ctx, ctxKey, id)

// 	s := &storage{}
// 	c.contextStorages.Store(id, s)

// 	go func() {
// 		<-ctx.Done()
// 		c.contextStorages.Delete(id)
// 		s.cleanup()
// 		s = nil
// 	}()

// 	return ctx
// }
