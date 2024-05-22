package di

import (
	"context"
)

type DisposerFunc func(any)

// Disposable interface to be implemented by components that want to release
// resources on destruction
//
// See Disposer
type Disposable interface {
	// Destroy invoked by the container on destruction of a component.
	Destroy()
}

// Disposer register a disposal function to the component. A factory
// component may declare multiple disposer methods. If the factory
// returns nil, the disposer will be ignored
//
// See Disposable
func Disposer[T any](disposer func(T)) FactoryConfig {
	return func(f *Factory) {
		f.disposers = append(f.disposers, func(a any) {
			if v, ok := a.(T); ok {
				disposer(v)
			}
		})
	}
}

// DisposableAdapter adapter that perform various destruction steps
// on a given component instance:
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
	if d.factory.HasDisposers() {
		for _, disposer := range d.factory.disposers {
			disposer(d.obj)
		}
	}
}
