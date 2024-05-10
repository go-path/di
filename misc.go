package di

import (
	"context"
	"reflect"
)

// Disposer A disposer allows the application to perform customized cleanup of an object returned by a producer.
type Disposer interface {
	Disposes()
}

var (
	_typeErr       = reflect.TypeOf((*error)(nil)).Elem()
	_typeNilReturn = reflect.TypeOf((*NilReturn)(nil)).Elem()
	_typeDisposer  = reflect.TypeOf((*Disposer)(nil)).Elem()
	_typeContext   = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func isError(t reflect.Type) bool {
	return t.Implements(_typeErr)
}

func isContext(t reflect.Type) bool {
	return t.Implements(_typeContext)
}
