package di

import "reflect"

type TypeBase[T any] struct {
}

// Type get the type of component
func (t TypeBase[T]) Type() reflect.Type {
	return Key[T]()
}

// Provider provides instances of T. For any type T that can be
// injected, you can also inject Provider<T>. Compared to
// injecting T directly, injecting Provider<T> enables:
//  - lazy or optional retrieval of an instance.
//  - breaking circular dependencies.
//
// Example:
//
//	di.Register(func(sp di.Provider[testService]) {
//		if condition {
//			service, err := sp.Get()
//		}
//	})
//
// See Unmanaged
type Provider[T any] struct {
	TypeBase[T]
	supplier func() (any, error)
}

// Get the value
func (p Provider[T]) Get() (o T, e error) {
	if v, err := p.supplier(); err != nil {
		e = err
	} else {
		o = v.(T)
	}
	return
}

func (p Provider[T]) With(supplier func() (any, error)) Provider[T] {
	return Provider[T]{TypeBase: TypeBase[T]{}, supplier: supplier}
}
