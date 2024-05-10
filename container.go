package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
)

type Container interface {
	Start(contexts ...context.Context) error
	AddContext(ctx context.Context) context.Context
	MustProvide(ctor any, opts ...ProviderOption)
	Provide(ctor any, opts ...ProviderOption) error
	RunOnStartup(ctor any, priority int, opts ...ProviderOption)
	Instances(ctx context.Context, valid func(p *Provider) bool, less func(a, b *Provider) bool) (instances []any, err error)
	Foreach(visitor func(p *Provider) (stop bool, err error)) error
	Get(key reflect.Type, contexts ...context.Context) (o any, e error)
	Cleanup()
	Clear()
	Mock(mock any) (cleanup func())
}

type container struct {
	graph            *graph
	providers        map[reflect.Type][]*Provider
	contextStorages  sync.Map
	singletonStorage *storage
	testingHasMock   bool
	testingMocks     map[reflect.Type]mockFn
	mockMu           sync.Mutex
}

func (c *container) Start(contexts ...context.Context) error {
	var ctx context.Context
	if len(contexts) > 0 {
		ctx = contexts[0]
	} else {
		ctx = context.Background()
	}

	c.Instances(ctx, func(p *Provider) bool {
		return p.IsStartup() && p.Scope() == SingletonScope
	}, func(a, b *Provider) bool {
		return a.StartupPriority() < b.StartupPriority()
	})

	return nil
}

func (c *container) AddContext(ctx context.Context) context.Context {
	id := ctxSeq.Add(1)
	ctx = context.WithValue(ctx, ctxKey, id)

	s := &storage{}
	c.contextStorages.Store(id, s)

	go func() {
		<-ctx.Done()
		c.contextStorages.Delete(id)
		s.cleanup()
		s = nil
	}()

	return ctx
}

func (c *container) MustProvide(ctor any, opts ...ProviderOption) {
	if err := c.Provide(ctor, opts...); err != nil {
		panic(err)
	}
}

// Provide
// func([context.Context], [dep A..N]) (A, [cleanup func()], [error])
func (c *container) Provide(ctor any, opts ...ProviderOption) error {
	ctorType := reflect.TypeOf(ctor)
	if ctorType == nil {
		return errors.New("can't createProvider an untyped nil")
	}

	var params []reflect.Type
	var returnType reflect.Type
	returnErrIdx := -1
	useContext := false

	if ctorType.Kind() != reflect.Func {
		returnType = ctorType
		ctorRef := ctor
		ctor = func() any {
			return ctorRef
		}
		ctorType = reflect.TypeOf(ctor)
	} else {
		// builds a params list from the provided constructor type.
		numArgs := ctorType.NumIn()
		if ctorType.IsVariadic() {
			numArgs--
		}

		params = make([]reflect.Type, 0, numArgs)

		for i := 0; i < numArgs; i++ {
			// @TODO: Qualifiers, like https://docs.oracle.com/javaee/6/api/javax/inject/Inject.html
			inType := ctorType.In(i)
			if isContext(inType) {
				if i == 0 {
					useContext = true
				} else {
					return fmt.Errorf("context.Context should be the first parameter in fucntion %v", ctorType)
				}
			}
			params = append(params, reflect.PointerTo(inType))
		}

		// results
		// func([context.Context], [dep A..N]) ([ServiceA], [error])
		numOut := ctorType.NumOut()
		if numOut == 0 {
			// used by initializers (Startup)
			returnType = _typeNilReturn
		} else {
			returnType = ctorType.Out(0)

			switch numOut {
			case 1:
				if isError(returnType) {
					// ex. "func() (error)"
					returnType = _typeNilReturn
					returnErrIdx = 0
				}
				break
			case 2:
				if isError(returnType) {
					// ex. "func() (error, Service)"
					return fmt.Errorf("%v has invalid return type (first) %v", ctorType, returnType)
				}

				secondType := ctorType.Out(1)
				if !isError(secondType) {
					// ex. "func() (ServiceA, ServiceB)"
					return fmt.Errorf("%v has invalid return type (second) %v", ctorType, secondType)
				}
				returnErrIdx = 1
				break
			default:
				return fmt.Errorf("%v has invalid return", ctorType)
			}
		}
	}

	key := reflect.PointerTo(returnType)

	p := &Provider{
		key:          key,
		scope:        SingletonScope,
		ctorValue:    reflect.ValueOf(ctor),
		ctorType:     ctorType,
		params:       params,
		useContext:   useContext,
		isDisposer:   returnType.Implements(_typeDisposer),
		returnType:   returnType,
		returnErrIdx: returnErrIdx,
		container:    c,
	}
	p.order = c.graph.add(p)

	for _, opt := range opts {
		opt.apply(p)
	}

	// cache old providers before running cycle detection.
	oldProviders := c.providers[key]
	c.providers[key] = append(c.providers[key], p)

	if len(c.providers[key]) > 1 {
		print("aqui")
	}

	if ok, cycle := c.graph.isAcyclic(); !ok {
		// When a cycle is detected, recover the old providers to reset
		// the providers map back to what it was before this node was
		// introduced.
		c.providers[key] = oldProviders
		fmt.Println(cycle)

		// , s.cycleDetectedError(cycle)
		return errors.New("this function introduces a cycle")
	}

	return nil
}

func (c *container) RunOnStartup(ctor any, priority int, opts ...ProviderOption) {
	c.MustProvide(ctor, append([]ProviderOption{Startup(priority)}, opts...)...)
}

func sort_default(a, b *Provider) bool {
	if a.IsPrimary() {
		return true
	} else if b.IsPrimary() {
		return false
	}

	if a.IsMock() != b.IsMock() {
		return b.IsMock()
	}

	if a.IsStartup() && b.IsStartup() {
		return a.StartupPriority() < b.StartupPriority()
	}

	return true
}

func (c *container) Instances(ctx context.Context, valid func(p *Provider) bool, less func(a, b *Provider) bool) (instances []any, err error) {

	var ps []*Provider

	for _, candidates := range c.providers {
		for _, p := range candidates {
			if valid(p) {
				ps = append(ps, p)
			}
		}
	}

	if less == nil {
		less = sort_default
	}

	sort.Slice(ps, func(i, j int) bool {
		return less(ps[i], ps[j])
	})

	for _, p := range ps {
		var e error
		var result *callResult

		if p.mock != nil {
			instances = append(instances, p.mock(ctx))
			continue
		}

		if result, e = p.call(ctx); e != nil {
			err = e
			return
		} else {
			// return
			err = result.err
			if err == nil && result.value != nil {
				instances = append(instances, result.value)
			}
		}
	}

	return
}

func (c *container) Foreach(visitor func(p *Provider) (stop bool, err error)) error {
	for _, candidates := range c.providers {
		for _, p := range candidates {
			stop, err2 := visitor(p)
			if err2 != nil {
				return err2
			}
			if stop {
				return nil
			}
		}
	}
	return nil
}

func (c *container) Get(key reflect.Type, contexts ...context.Context) (o any, e error) {
	var ctx context.Context
	if len(contexts) > 0 {
		ctx = contexts[0]
	} else {
		ctx = context.Background()
	}

	var p *Provider

	p, e = c.getProvider(key)
	if e != nil {
		return
	}
	if p.mock != nil {
		o = p.mock(ctx)
		return
	}

	var s *storage

	if p.scope == SingletonScope {
		s = c.singletonStorage
	} else if p.scope == ContextScope {
		if ss, ok := c.getStore(ctx); !ok {
			e = fmt.Errorf("o getProvider %v requer a inicialização do contexto", p)
			return
		} else {
			s = ss
		}
	}

	if s != nil {
		// first check if the storage already has cached a value for the type.
		if result, ok := s.get(key); ok {
			o = result.value
			return
		}
	}

	// callResult getProvider
	if result, errCall := p.call(ctx); errCall != nil {
		e = errCall
	} else {
		// return
		e = result.err
		if e == nil && result.value != nil {
			o = result.value
			if s != nil {
				s.set(key, result)
			}
		}
	}
	return
}

func (c *container) Cleanup() {
	c.contextStorages.Range(func(key, value any) bool {
		c.contextStorages.Delete(key)
		if s, ok := value.(*storage); ok {
			s.cleanup()
		}
		return true
	})
	c.singletonStorage.cleanup()
}

func (c *container) Clear() {
	c.Cleanup()
	c.graph = &graph{container: c}
	c.providers = make(map[reflect.Type][]*Provider)
	c.singletonStorage = &storage{}
	c.testingHasMock = false
	c.testingMocks = make(map[reflect.Type]mockFn)
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

func (c *container) getStore(ctx context.Context) (*storage, bool) {
	if id, ok := ctx.Value(ctxKey).(uint64); !ok {
		return nil, false
	} else if s, sok := c.contextStorages.Load(id); !sok {
		return nil, false
	} else {
		return s.(*storage), true
	}
}

func (c *container) getProvider(key reflect.Type) (p *Provider, e error) {

	if c.testingHasMock { // see Mock
		if fn, ok := c.testingMocks[key]; ok {
			return &Provider{mock: fn}, nil
		}
	}

	// Get candidates
	candidates := c.providers[key]

	switch len(candidates) {
	case 0:
		e = fmt.Errorf("%v não foi encontrado candidato para o tipo", key)
	case 1:
		p = candidates[0]
	default:
		// check for primary
		for _, candidate := range candidates {
			if candidate.isPrimary {
				p = candidate
				break
			}
		}
		if p == nil {
			e = fmt.Errorf("%v multiplos candidatos para o tipo", key)
		}
	}

	return
}
