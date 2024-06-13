package di

import (
	"strings"
)

// FactoryConfig is the type to configure the component Factory.
// Container.Register accepts any number of config (this is functional option pattern).
type FactoryConfig func(*Factory)

// Singleton identifies a component that only instantiates once.
//
// Example:
//
//	di.Register(func() MyService {
//		return &MyServiceImpl{ Id: uuid.New() }
//	}, di.Singleton)
//
//	di.Register(func(s MyService) MyControllerA {
//		print(s.Id) // uuid value
//	})
//
//	di.Register(func(s MyService) MyControllerB {
//		print(s.Id) // same uuid value
//	})
func Singleton(f *Factory) {
	f.scope = SCOPE_SINGLETON
}

// Prototype identifies a component that a new instance is created
// every time the component factory is invoked.
//
// Example:
//
//	di.Register(func() MyService {
//		return &MyServiceImpl{ Id: uuid.New() }
//	}, di.Prototype)
//
//	di.Register(func(s MyService) MyControllerA {
//		print(s.Id) // first uuid
//	})
//
//	di.Register(func(s MyService, ctn di.Container, ctx context.Context) MyControllerB {
//		print(s.Id) // second uuid
//
//		s2, _ := di.Get[testService](ctn, ctx)
//		print(s2.Id) // third uuid
//	})
func Prototype(f *Factory) {
	f.scope = SCOPE_PROTOTYPE
}

// Scoped identifies the lifecycle of an instance, such as singleton,
// prototype, and so forth.. A scope governs how the container
// reuses instances of the type.
//
// To register additional custom scopes, see Container.RegisterScope.
//
// Defaults to an empty string ("") which implies SCOPE_SINGLETON.
func Scoped(scope string) FactoryConfig {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = SCOPE_SINGLETON
	}
	return func(f *Factory) {
		f.scope = scope
	}
}

// Startup indicates that this component must be initialized during
// container initialization (Container.Initialize method)
//
// Example:
//
//	di.Register(func()  {
//		print("Second")
//	}, Startup(200))
//
//	di.Register(func()  {
//		print("First")
//	}, Startup(100))
func Startup(order int) FactoryConfig {
	return func(f *Factory) {
		f.order = order
		f.startup = true
	}
}

// Order can be applied to any component to indicate in what order they
// should be used.
//
// Higher values are interpreted as lower priority. As a consequence,
// the object with the lowest value has the highest priority.
//
// Same order values will result in arbitrary sort positions for the
// affected objects.
//
// If the component is marked as Startup, the order determines its
// execution order.
//
// Order is also used during dependency injection. The candidate with the
// lower order will be injected.
//
// A framework can implement filters and use order to define the order
// of execution
func Order(order int) FactoryConfig {
	return func(f *Factory) {
		f.order = order
	}
}

// Stereotype a stereotype encapsulates any combination of ComponentOption
//
// Example:
//
//	var Controller = di.Stereotype(di.Singleton, di.Qualify[testQualifier](), di.Startup(500))
//
//	di.Register(func() MyController {
//		return &MyController{}
//	}, Controller)
//
// Example: Filter using Stereotype
//
//	di.Filter(Controller).Foreach(func(f *Factory) (bool, error) { ... })
func Stereotype(options ...FactoryConfig) FactoryConfig {
	return func(f *Factory) {
		for _, option := range options {
			option(f)
		}
	}
}

// Condition a single condition that must be matched in order for a component to be registered.
// Conditions are checked immediately before the component factory is due to be
// registered and are free to veto registration based on any criteria
// that can be determined at that point.
func Condition(condition ConditionFunc) FactoryConfig {
	return func(f *Factory) {
		f.conditions = append(f.conditions, condition)
	}
}

// @TODO:
// Lazy, Whether lazy initialization should occur.
// DependsOn[AnoterService]()
// PreDestroy(T)
