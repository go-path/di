package di

import (
	"context"
	"reflect"
)

var global = New()

func New() Container {
	//providers        = map[reflect.Type][]*Provider{}
	//contextStorages  = sync.Map{}
	//singletonStorage = &storage{}
	//testingHasMock   = false
	//testingMocks     = map[reflect.Type]mockFn{} // see Mock
	//contextStorages:  sync.Map{},
	c := &container{
		graph:            &graph{},
		providers:        make(map[reflect.Type][]*Provider),
		singletonStorage: &storage{},
		testingHasMock:   false,
		testingMocks:     make(map[reflect.Type]mockFn),
	}
	c.graph.container = c
	return c
}

func Start(contexts ...context.Context) error {
	return global.Start(contexts...)
}

func AddContext(ctx context.Context) context.Context {
	return global.AddContext(ctx)
}

func MustProvide(ctor any, opts ...ProviderOption) {
	global.MustProvide(ctor, opts...)
}

func Provide(ctor any, opts ...ProviderOption) error {
	return global.Provide(ctor, opts...)
}

func RunOnStartup(ctor any, priority int, opts ...ProviderOption) {
	global.RunOnStartup(ctor, priority, opts...)
}

func Instances(ctx context.Context, valid func(p *Provider) bool, less func(a, b *Provider) bool) (instances []any, err error) {
	return global.Instances(ctx, valid, less)
}

func Foreach(visitor func(p *Provider) (stop bool, err error)) error {
	return global.Foreach(visitor)
}

func MustGet[T any](contexts ...context.Context) T {
	o, err := Get[T](contexts...)
	if err != nil {
		panic(err)
	}
	return o
}

func Get[T any](contexts ...context.Context) (o T, e error) {
	var instance *T
	key := reflect.TypeOf(instance)

	if v, err := global.Get(key, contexts...); err != nil {
		e = err
	} else {
		o = v.(T)
	}
	return
}

func Key[T any]() reflect.Type {
	var instance *T
	return reflect.TypeOf(instance)
}

func KeyOf(t any) reflect.Type {
	if tt, ok := t.(reflect.Type); ok {
		return reflect.PointerTo(tt)
	} else if tt, ok := t.(reflect.Value); ok {
		return reflect.PointerTo(tt.Type())
	}
	return reflect.PointerTo(reflect.TypeOf(t))
}

func Global() Container {
	return global
}

func Cleanup() {
	global.Cleanup()
}

func Clear() {
	global.Clear()
}

func Mock(mock any) (cleanup func()) {
	return global.Mock(mock)
}
