package di

import (
	"context"
	"reflect"
)

var global = New(nil)

func Global() Container {
	return global
}

// Initialize initialize all non-lazy singletons (startup)
func Initialize(ctx ...context.Context) error {
	return global.Initialize(ctx...)
}

func Register(ctor any, opts ...FactoryConfig) {
	global.Register(ctor, opts...)
}

func ShouldRegister(ctor any, opts ...FactoryConfig) error {
	return global.ShouldRegister(ctor, opts...)
}

// RegisterScope Register the given scope, backed by the given ScopeI implementation.
func RegisterScope(name string, scope ScopeI) error {
	return global.RegisterScope(name, scope)
}

// Get return an instance, which may be shared or independent, of the specified component.
func Get[T any](ctx ...context.Context) (o T, e error) {
	return GetFrom[T](global, ctx...)
}

// Contains check if this container contains a component with the given key.
// Does not consider any hierarchy this container may participate in.
func Contains(key reflect.Type) bool {
	return global.Contains(key)
}

func Filter(options ...FactoryConfig) *FilteredFactories {
	return global.Filter(options...)
}

func GetObjectFactory(factory *Factory, managed bool, ctx ...context.Context) ObjectFactory {
	return global.GetObjectFactory(factory, managed, ctx...)
}

func GetObjectFactoryFor(key reflect.Type, managed bool, ctx ...context.Context) ObjectFactory {
	return global.GetObjectFactoryFor(key, managed, ctx...)
}

// ResolveArgs returns an ordered list of values which may be passed directly to the Factory Create method
func ResolveArgs(factory *Factory, ctx ...context.Context) ([]reflect.Value, error) {
	return global.ResolveArgs(factory, ctx...)
}

// Destroy this container
func Destroy() error {
	return global.Destroy()
}

// DestroyObject destroy the given instance
func DestroyObject(key reflect.Type, object any) error {
	return global.DestroyObject(key, object)
}

// DestroySingletons destroy all singleton components in this container. To be called on shutdown of a factory.
func DestroySingletons() error {
	return global.DestroySingletons()
}

// Mock test only, register a mock instance to the container
func Mock(mock any) (cleanup func()) {
	return global.Mock(mock)
}
