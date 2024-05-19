package di

import (
	"reflect"
)

// ComponentConfig is the type to replace default parameters.
// di.Provide accepts any number of options (this is functional option pattern).
type ComponentConfig func(*Factory)

func Primary(f *Factory) {
	f.primary = true
}

func Singleton(f *Factory) {
	f.scope = SCOPE_SINGLETON
}

func Prototype(f *Factory) {
	f.scope = SCOPE_PROTOTYPE
}

// Scoped set the scope of the component
func Scoped(scope string) ComponentConfig {
	return func(f *Factory) {
		f.scope = scope
	}
}

// Startup indicates that this component must be initialized during container initialization
func Startup(priority int) ComponentConfig {
	return func(f *Factory) {
		f.startup = true
		f.startupPriority = priority
	}
}

// Disposer register a disposal function to the component.
func Disposer[T any](disposer func(T)) ComponentConfig {
	return func(f *Factory) {
		f.disposers = append(f.disposers, reflect.ValueOf(disposer))
	}
}

// Disposer register a disposal function to the component.
func PostConstruct[T any](callback func(T)) ComponentConfig {
	return func(f *Factory) {
		f.disposers = append(f.disposers, reflect.ValueOf(callback))
	}
}

// Priority can be applied to any component to indicate in what order they should be used.
// The effect of using the Priority in any particular instance is defined by other specifications.
// Ex. A framework can implement filters and use priority to define the order of execution
func Priority(priority int) ComponentConfig {
	return func(f *Factory) {
		f.priority = priority
	}
}

// Stereotype a stereotype encapsulates any combination of ComponentOption
func Stereotype(options ...ComponentConfig) ComponentConfig {
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
func Condition(condition ConditionFn) ComponentConfig {
	return func(f *Factory) {
		f.conditions = append(f.conditions, condition)
	}
}

// @TODO:
// Lazy, Whether lazy initialization should occur.
// PreDestroy(T)
