package di

import (
	"context"
)

var global = New(nil)

func Global() Container {
	return global
}

func Register(ctor any, opts ...ComponentConfig) {
	global.Register(ctor, opts...)
}

func ShouldRegister(ctor any, opts ...ComponentConfig) error {
	return global.ShouldRegister(ctor, opts...)
}

func Initialize(contexts ...context.Context) error {
	return global.Initialize(contexts...)
}

func Mock(mock any) (cleanup func()) {
	return global.Mock(mock)
}

// func AddContext(ctx context.Context) context.Context {
// 	return global.AddContext(ctx)
// }

// func RunOnStartup(ctor any, priority int, opts ...ComponentOption) {
// 	global.RunOnStartup(ctor, priority, opts...)
// }

// func Instances(ctx context.Context, valid func(p *Factory) bool, less func(a, b *Factory) bool) (instances []any, err error) {
// 	return global.Instances(ctx, valid, less)
// }

// func Foreach(visitor func(p *Factory) (stop bool, err error)) error {
// 	return global.Foreach(visitor)
// }
