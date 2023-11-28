package di

import (
	"context"
	"reflect"
)

type Scope uint8

const (
	SingletonScope Scope = iota
	PrototypeScope
	ContextScope
)

var (
	_errType = reflect.TypeOf((*error)(nil)).Elem()
	_ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func isError(t reflect.Type) bool {
	return t.Implements(_errType)
}

func isContext(t reflect.Type) bool {
	return t.Implements(_ctxType)
}
