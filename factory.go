package di

import (
	"context"
	"reflect"
)

// nilReturn internal representation of a nil component instance, e. g. for a nil value returned from Factory
type nilReturn struct{}

type mockFn func(ctx context.Context) any

type ConditionFn func(Container, *Factory) bool

// InitializerComponent interface to be implemented by components that want to execute callback on creation
type InitializerComponent interface {
	Initialize()
}

// DisposableComponent interface to be implemented by components that want to release resources on destruction
type DisposableComponent interface {
	// Destroy invoked by the container on destruction of a component.
	Destroy()
}

// Factory is a node in the dependency graph that represents a constructor provided by the user
type Factory struct {
	order           int                   // order of this node in graph
	key             reflect.Type          // key for this factory
	factoryType     reflect.Type          // type information about constructor
	factoryValue    reflect.Value         // constructor function
	returnType      reflect.Type          // type information about return type.
	returnErrorIdx  int                   // error return index (-1, 0 or 1)
	returnValueIdx  int                   // value return index (0 or 1)
	parameterKeys   []reflect.Type        // type information about factory parameters.
	scope           string                // Factory scope
	primary         bool                  // defines a preference when multiple providers of the same type are present
	priority        int                   // priority of use
	startup         bool                  // will be initialized with container
	startupPriority int                   // priority of initialization
	initializers    []reflect.Value       // post construct callbacks
	disposers       []reflect.Value       // disposal functions
	conditions      []ConditionFn         // indicates that a component is only eligible for registration when all specified conditions match.
	qualifiers      map[reflect.Type]bool // component qualifiers
	mock            mockFn
}

// Create a new instance of component.
func (f *Factory) Create(args []reflect.Value) (any, error) {

	results := f.factoryValue.Call(args)

	if f.ReturnsError() {
		if err := results[f.returnErrorIdx]; !err.IsNil() {
			return nil, err.Interface().(error)
		}
	}

	if !f.ReturnsValue() {
		return nil, nil
	}

	return results[f.returnValueIdx].Interface(), nil
}

func (f *Factory) Singleton() bool {
	return f.scope == SCOPE_SINGLETON
}

func (f *Factory) Scope() string {
	return f.scope
}

func (f *Factory) Primary() bool {
	return f.primary
}

func (f *Factory) Mock() bool {
	return f.mock != nil
}

func (f *Factory) Constructor() reflect.Value {
	return f.factoryValue
}

func (f *Factory) Params() []reflect.Type {
	return f.parameterKeys[0:]
}

func (f *Factory) Type() reflect.Type {
	return f.returnType
}

func (f *Factory) ReturnsError() bool {
	return f.returnErrorIdx != -1
}

func (f *Factory) ReturnsValue() bool {
	return f.returnValueIdx != -1
}

func (f *Factory) ReturnErrorIdx() int {
	return f.returnErrorIdx
}

func (f *Factory) ReturnValueIdx() int {
	return f.returnValueIdx
}

func (f *Factory) Disposer() bool {
	return len(f.disposers) > 0
}

func (f *Factory) Startup() bool {
	return f.startup
}

func (f *Factory) StartupPriority() int {
	return f.startupPriority
}

func (f *Factory) Priority() int {
	return f.priority
}

// func (f *Factory) Load(contexts ...context.Context) error {
// 	_, err := f.container.Get(f.key, contexts...)
// 	return err
// }

// func (f *Factory) Get(contexts ...context.Context) (any, error) {
// 	return f.container.Get(f.key, contexts...)
// }

// Destruction

// func (p *Factory) UseContext() bool {
// 	return p.useContext
// }

// func (p *Factory) instantiate(contexts ...context.Context) (o any, e error) {
// 	var ctx context.Context
// 	if len(contexts) > 0 {
// 		ctx = contexts[0]
// 	} else {
// 		ctx = context.Background()
// 	}

// 	if p.mock != nil {
// 		o = p.mock(ctx)
// 		return
// 	}

// 	return p.Create(ctx)
// }

// Parameter 2 of constructor in yanbin.blog.testweb.controllers.ManagementController required a bean of type 'yanbin.blog.testweb.service.CalcEngineFactory' that could not be found
// org.springframework.beans.factory.NoSuchBeanDefinitionException: No qualifying bean of type [in.amruth.xplore.utility.IUtil] found for dependency: expected at least 1 bean which qualifies as autowire candidate for this dependency. Dependency annotations: {@org.springframework.beans.factory.annotation.Autowired(required=true)}
// Field dependency in com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanA required a bean of type 'com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanB' that could not be found.
// No qualifying bean of type
//  [com.baeldung.packageB.IBeanB] is defined:
//expected single matching bean but found 2: beanB1,beanB2
// https://www.baeldung.com/spring-nosuchbeandefinitionexception
