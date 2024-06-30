package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var ErrNotStruct = errors.New("the Injected method only accepts struct or *struct")

// Injected simplifies component registration through reflection.
//
// Example:
//
//	type myController struct {
//		MyService Service `inject:""`
//	}
//
//	di.Register(di.Injected[*myController]())
//
// In the example above, the MyService dependency will be injected automatically.
func Injected[T any]() func(Container, context.Context) (out T, err error) {
	otp := reflect.TypeOf((*T)(nil)).Elem()
	otp_nptr := otp
	is_ptr := (otp.Kind() == reflect.Pointer)
	if is_ptr {
		otp_nptr = otp.Elem()
	}

	if otp_nptr.Kind() != reflect.Struct {
		panic(ErrNotStruct)
	}

	var depsFieldIdx []int
	var depsFieldKey []reflect.Type

	for fieldIndex := 0; fieldIndex < otp_nptr.NumField(); fieldIndex++ {
		field := otp_nptr.Field(fieldIndex)
		if !field.IsExported() {
			continue
		}
		if _, hasTag := field.Tag.Lookup("inject"); !hasTag {
			continue
		}

		depsFieldIdx = append(depsFieldIdx, fieldIndex)
		depsFieldKey = append(depsFieldKey, KeyOf(field.Type))
	}

	return func(ctn Container, ctx context.Context) (out T, err error) {
		nptr_ptr := reflect.New(otp_nptr) // Pointer Struct
		nptr_val := nptr_ptr.Elem()       // Value  Struct

		for i, fieldIndex := range depsFieldIdx {
			// resolve dependency
			depk := depsFieldKey[i]
			if dep, e := ctn.Get(depk, ctx); e != nil {
				err = errors.Join(fmt.Errorf(`cannot resolve dependency "%s" for "%s"`, depk.String(), otp.String()), e)
				return
			} else {
				nptr_val.Field(fieldIndex).Set(reflect.ValueOf(dep))
			}
		}

		if is_ptr {
			// interface {*Struct}
			out = (nptr_val.Addr().Interface()).(T)
		} else {
			// interface {struct}
			out = (nptr_val.Interface()).(T)
		}

		return
	}
}
