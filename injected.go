package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var ErrNotStruct = errors.New("the Injected method only accepts struct or *struct")

// Injector simplifies component registration through reflection.
//
// Example:
//
//	type myController struct {
//		MyService Service `inject:""`
//	}
//
//	di.Register(di.Injector[*myController]())
//
// In the example above, the MyService dependency will be injected automatically.
//
// @TODO:  embedded structs
func Injector[T any]() func(Container, context.Context) (out T, err error) {
	injector := InjectorOf(reflect.TypeOf((*T)(nil)).Elem())
	return func(ctn Container, ctx context.Context) (out T, err error) {
		var o any
		if o, err = injector(ctn, ctx); err == nil {
			out = o.(T)
		}
		return
	}
}

type IntectorFn func(Container, context.Context) (out any, err error)

var (
	injectors = map[reflect.Type]IntectorFn{}
)

func InjectorOf(structType reflect.Type) IntectorFn {
	structTypeNoPtr := structType
	isPointer := (structType.Kind() == reflect.Pointer)
	if isPointer {
		structTypeNoPtr = structType.Elem()
	}

	if structTypeNoPtr.Kind() != reflect.Struct {
		panic(ErrNotStruct)
	}

	if injector, exists := injectors[structType]; exists {
		return injector
	}

	var depsFieldIdx []int
	var depsFieldKey []reflect.Type

	for fieldIndex := 0; fieldIndex < structTypeNoPtr.NumField(); fieldIndex++ {
		field := structTypeNoPtr.Field(fieldIndex)
		if !field.IsExported() {
			continue
		}

		if _, hasTag := field.Tag.Lookup("inject"); !hasTag {
			continue
		}

		depsFieldIdx = append(depsFieldIdx, fieldIndex)
		depsFieldKey = append(depsFieldKey, KeyOf(field.Type))
	}

	injector := func(ctn Container, ctx context.Context) (out any, err error) {
		nptr_ptr := reflect.New(structTypeNoPtr) // Pointer Struct
		nptr_val := nptr_ptr.Elem()              // Value  Struct

		for i, fieldIndex := range depsFieldIdx {
			// resolve dependency
			depk := depsFieldKey[i]
			if dep, e := ctn.Get(depk, ctx); e != nil {
				// automatically inject Struct (prototype scoped)
				if errors.Is(e, ErrCandidateNotFound) {
					st := depk
					if st.Kind() == reflect.Pointer {
						st = st.Elem()
					}
					if st.Kind() == reflect.Struct {
						injector := InjectorOf(depk)
						if dep, ierr := injector(ctn, ctx); ierr != nil {
							e = ierr
						} else {
							// instance created - initializers/post construct
							if i, ok := dep.(Initializable); ok {
								i.Initialize()
							}

							nptr_val.Field(fieldIndex).Set(reflect.ValueOf(dep))
							continue
						}
					}
				}

				err = errors.Join(fmt.Errorf(`cannot resolve dependency "%s" for "%s"`, depk.String(), structType.String()), e)
				return
			} else {
				nptr_val.Field(fieldIndex).Set(reflect.ValueOf(dep))
			}
		}

		if isPointer {
			// interface {*Struct}
			out = nptr_val.Addr().Interface()
		} else {
			// interface {struct}
			out = nptr_val.Interface()
		}

		return
	}
	injectors[structType] = injector
	return injector
}
