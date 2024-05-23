package di

import (
	"context"
	"reflect"
)

const (
	SCOPE_SINGLETON string = "singleton"
	SCOPE_PROTOTYPE string = "prototype"
)

type CreateObjectFunc func() (any, DisposableAdapter, error)

type ScopeI interface {
	// Get return the object with the given key from the underlying scope,
	// creating it if not found in the underlying storage mechanism.
	//
	// If CreateObjectFunc returns a disposer, the scope need to register a callback to
	// be executed on destruction of the specified object in the scope (or at
	// destruction of the entire scope, if the scope does not destroy individual
	//	objects but rather only terminates in its entirety).
	Get(context.Context, reflect.Type, *Factory, CreateObjectFunc) (any, error)

	// Remove the object with the given key from the underlying scope.
	// Returns nil if no object was found; otherwise returns the removed Object.
	Remove(reflect.Type, any) (any, error)

	Destroy()
}

type scopePrototypeImpl struct {
}

func (s *scopePrototypeImpl) Get(ctx context.Context, key reflect.Type, factory *Factory, createObject CreateObjectFunc) (any, error) {
	obj, _, err := createObject()
	return obj, err
}

func (s *scopePrototypeImpl) Remove(reflect.Type, any) (any, error) {
	return nil, nil
}

func (s *scopePrototypeImpl) Destroy() {

}
