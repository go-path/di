package di

import "reflect"

// Initializable interface to be implemented by components that want to
// initialize resources on creation
//
// See Initializer
type Initializable interface {
	// Initialize invoked by the container on creation of a component.
	Initialize()
}

// Initializer register a initializer function to the component. A factory
// component may declare multiple initializers methods. If the factory
// returns nil, the initializer will be ignored
//
// See Initializable
func Initializer[T any](callback func(T)) FactoryConfig {
	return func(f *Factory) {
		f.initializers = append(f.initializers, reflect.ValueOf(callback))
	}
}
