package di

import (
	"context"
	"reflect"
)

type DisposableAdapter interface {
	Context() context.Context
	Dispose()
}

type disposableAdapterImpl struct {
	obj        any
	factory    *Factory
	container  Container
	getContext func() context.Context
}

func (d *disposableAdapterImpl) Context() context.Context {
	return d.getContext()
}

func (d *disposableAdapterImpl) Dispose() {
	if d.factory.Disposer() {
		preDestroyArgs := []reflect.Value{reflect.ValueOf(d.obj)}
		for _, disposer := range d.factory.disposers {
			disposer.Call(preDestroyArgs)
		}
	}
}
