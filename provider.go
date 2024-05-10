package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type callResult struct {
	err        error
	value      any
	isDisposer bool
}

type Scope uint8

const (
	SingletonScope Scope = iota
	PrototypeScope
	ContextScope
)

type NilReturn struct{}

// Provider is a node in the dependency graph that represents a constructor
// provided by the user
type Provider struct {
	key             reflect.Type
	order           int            // order of this node in graph
	scope           Scope          // Provider scope
	isPrimary       bool           // defines a preference when multiple providers of the same type are present
	ctorValue       reflect.Value  // constructor function
	ctorType        reflect.Type   // type information about constructor
	params          []reflect.Type // type information about ctorValue parameters.
	useContext      bool           // first param is context.Context
	returnType      reflect.Type   // type information about ctorValue return.
	returnErrIdx    int            // error return index (-1, 0 or 1)
	isStartup       bool           // will be initialized with container
	startupPriority int            // priority of initialization
	isDisposer      bool           // returnType is a Disposer
	mock            mockFn
	container       *container
}

func (p *Provider) Load(contexts ...context.Context) error {
	_, err := p.container.Get(p.key, contexts...)
	return err
}

func (p *Provider) Get(contexts ...context.Context) (any, error) {
	return p.container.Get(p.key, contexts...)
}

func (p *Provider) Scope() Scope {
	return p.scope
}

func (p *Provider) IsPrimary() bool {
	return p.isPrimary
}

func (p *Provider) IsMock() bool {
	return p.mock != nil
}

func (p *Provider) Constructor() reflect.Value {
	return p.ctorValue
}

func (p *Provider) Params() []reflect.Type {
	return p.params[0:]
}

func (p *Provider) Type() reflect.Type {
	return p.returnType
}

func (p *Provider) ReturnErrIdx() int {
	return p.returnErrIdx
}

func (p *Provider) IsDisposer() bool {
	return p.isDisposer
}

func (p *Provider) UseContext() bool {
	return p.useContext
}

func (p *Provider) IsStartup() bool {
	return p.isStartup
}

func (p *Provider) StartupPriority() int {
	return p.startupPriority
}

func (p *Provider) instantiate(contexts ...context.Context) (o any, e error) {
	var ctx context.Context
	if len(contexts) > 0 {
		ctx = contexts[0]
	} else {
		ctx = context.Background()
	}

	if p.mock != nil {
		o = p.mock(ctx)
		return
	}

	return p.call(ctx)
}

func (p *Provider) call(ctx context.Context) (out *callResult, callError error) {

	if err := p.shallowCheckDependencies(); err != nil {
		callError = errors.Join(fmt.Errorf("missing dependencies for function %v", p.ctorType), err)
		return
	}

	if args, pErr := p.getParams(ctx); pErr != nil {
		callError = pErr
	} else {
		out = &callResult{isDisposer: p.isDisposer}
		results := p.ctorValue.Call(args)

		if p.returnErrIdx != -1 {
			cErr := results[p.returnErrIdx]
			if !cErr.IsNil() {
				out.err = cErr.Interface().(error)
			}
		}

		if out.err == nil {
			out.value = results[0].Interface()
		}
	}

	// results := c.invoker()(reflect.ValueOf(n.ctor), args)
	return
}

// BuildList returns an ordered list of values which may be passed directly
// to the underlying ctorValue.
func (p *Provider) getParams(ctx context.Context) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(p.params))
	for i, param := range p.params {
		if p.useContext && i == 0 {
			args[0] = reflect.ValueOf(ctx)
			continue
		}
		v, err := p.container.Get(param, ctx)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(v)
	}
	return args, nil
}

// Checks that all direct dependencies of the provided parameters are present in
// the container. Returns an error if not.
func (p *Provider) shallowCheckDependencies() error {
	var err error

	missingDeps := p.findMissingDependencies()
	for _, dep := range missingDeps {
		err = errors.Join(err, fmt.Errorf("\t%v", dep))
	}

	return err
}

func (p *Provider) findMissingDependencies() []reflect.Type {
	var missingDeps []reflect.Type

	for i, param := range p.params {
		if p.useContext && i == 0 {
			// ignore context
			continue
		}
		allProviders := p.container.providers[param]
		// This means that there is no getProvider that provides this value,
		// and it is NOT being decorated and is NOT optional.
		// In the case that there is no providers but there is a decorated value
		// of this type, it can be provided safely so we can safely skip this.
		if len(allProviders) == 0 {
			if p.container.testingHasMock { // see Mock
				if _, ok := p.container.testingMocks[param]; ok {
					continue
				}
			}

			missingDeps = append(missingDeps, param)
		}
	}
	return missingDeps
}

// Parameter 2 of constructor in yanbin.blog.testweb.controllers.ManagementController required a bean of type 'yanbin.blog.testweb.service.CalcEngineFactory' that could not be found
// org.springframework.beans.factory.NoSuchBeanDefinitionException: No qualifying bean of type [in.amruth.xplore.utility.IUtil] found for dependency: expected at least 1 bean which qualifies as autowire candidate for this dependency. Dependency annotations: {@org.springframework.beans.factory.annotation.Autowired(required=true)}
// Field dependency in com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanA required a bean of type 'com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanB' that could not be found.
// No qualifying bean of type
//  [com.baeldung.packageB.IBeanB] is defined:
//expected single matching bean but found 2: beanB1,beanB2
// https://www.baeldung.com/spring-nosuchbeandefinitionexception
