package di

import (
	"context"
	"reflect"
)

var (
	_typeErr         = Key[error]()
	_keyContext      = Key[context.Context]()
	_keyContainer    = Key[Container]()
	_typeNilReturn   = Key[nilReturn]()
	_typeDisposable  = Key[Disposable]()
	_typeInitializer = Key[Initializable]()
	_typeReflectType = Key[reflect.Type]()
)

func isError(t reflect.Type) bool {
	return t.Implements(_typeErr)
}

func getContext(contexts ...context.Context) (ctx context.Context) {
	if len(contexts) > 0 {
		ctx = contexts[0]
	}

	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// Key is a pointer to a type
func Key[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// KeyOf get a key for a value
func KeyOf(t any) reflect.Type {
	if tt, ok := t.(reflect.Type); ok {
		return tt
	} else if tt, ok := t.(reflect.Value); ok {
		return tt.Type()
	}
	return reflect.TypeOf(t)
}

// GetFrom get a instance from container using generics (returns error)
func GetFrom[T any](c Container, contexts ...context.Context) (o T, e error) {
	if v, err := c.Get(Key[T](), contexts...); err != nil {
		e = err
	} else {
		o = v.(T)
	}
	return
}

// MustGetFrom get a instance from container using generics (panic on error)
func MustGetFrom[T any](c Container, ctx ...context.Context) T {
	o, err := GetFrom[T](c, ctx...)
	if err != nil {
		panic(err)
	}
	return o
}

func FilterOf[T any](c Container) *FilteredFactories {
	key := Key[T]()
	cond := Condition(func(c Container, f *Factory) bool {
		return f.key == key || f.Type().AssignableTo(key)
	})

	return c.Filter(cond)
}

func AllOf[T any](c Container, ctx context.Context) (o []T, e error) {
	return AllOfFilter[T](FilterOf[T](c), ctx)
}

func AllOfFilter[T any](filter *FilteredFactories, ctx context.Context) (o []T, e error) {
	var objects []T

	err := filter.Foreach(func(f *Factory) (bool, error) {
		if obj, disposer, err := filter.container.GetObjectFactory(f, true, ctx)(); err != nil {
			return true, err
		} else if o, ok := obj.(T); ok {
			objects = append(objects, o)
		} else if disposer != nil {
			disposer.Dispose()
		}
		return false, nil
	})

	return objects, err
}
