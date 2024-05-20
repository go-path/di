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

// func isContext(t reflect.Type) bool {
// 	return t == _keyContext
// }

func getContext(contexts ...context.Context) context.Context {
	if len(contexts) > 0 {
		return contexts[0]
	} else {
		return context.Background()
	}
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

func Get[T any](ctn Container, contexts ...context.Context) (o T, e error) {
	if v, err := ctn.Get(Key[T](), contexts...); err != nil {
		e = err
	} else {
		o = v.(T)
	}
	return
}

func MustGet[T any](ctn Container, contexts ...context.Context) T {
	o, err := Get[T](ctn, contexts...)
	if err != nil {
		panic(err)
	}
	return o
}

func AllOf[T any](c Container, ctx context.Context) (o []T, e error) {
	key := Key[T]()
	// t := reflect.TypeOf((*T)(nil)).Elem()

	cond := Condition(func(c Container, f *Factory) bool {
		return f.key == key || f.Type().AssignableTo(key)
	})

	var objects []T

	err := c.Filter(cond).Foreach(func(f *Factory) (bool, error) {
		if obj, disposer, err := c.GetObjectFactory(f, true, ctx)(); err != nil {
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
