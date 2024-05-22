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
func Startup(priority int) FactoryConfig {
	return func(f *Factory) {
		f.startup = true
		f.priority = priority
	}
}

// Priority can be applied to any component to indicate in what order they should be used.
//
// If the component is marked as Startup, the priority determines its execution order.
//
// Priority is also used during dependency injection. The candidate with the
// highest priority will be injected.
//
// A framework can implement filters and use priority to define the order of execution
func Priority(priority int) FactoryConfig {
	return func(f *Factory) {
		f.priority = priority
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
