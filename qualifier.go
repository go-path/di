package di

import (
	"reflect"
)

// Qualified allows you to inject a dependency that has a qualifier
//
// Example:
//
//	type MyQualifier string
//
//	di.Register(func(sq Qualified[*MyService, MyQualifier]) {
//		myService := sq.Get()
//	})
type Qualified[T any, Q any] struct {
	TypeBase[T]
	value any
}

// Get the value
func (q Qualified[T, Q]) Get() T {
	return q.value.(T)
}

// Type get the type of component
// func (q Qualified[T, Q]) Type() reflect.Type {
// 	return Key[T]()
// }

// Qualifier get the qualifier type
func (q Qualified[T, Q]) Qualifier() reflect.Type {
	return Key[Q]()
}

// WithValue create a new instance of Qualified with the value
func (q Qualified[T, Q]) With(value any) Qualified[T, Q] {
	return Qualified[T, Q]{TypeBase: TypeBase[T]{}, value: value}
}

// Qualify register a qualifier for the component. Anyone can define a new qualifier.
//
// Example:
//
//	type MyQualifier string
//
//	di.Register(func() *MyService {
//		return &MyService{}
//	}, di.Qualify[testQualifier]())
func Qualify[Q any]() FactoryConfig {
	qualifier := Key[Q]()
	return func(f *Factory) {
		f.qualifiers[qualifier] = true
	}
}

type PrimaryQualifier uint8

var (
	_primaryQualifierKey  = Key[PrimaryQualifier]()
	_primaryQualifyConfig = Qualify[PrimaryQualifier]()
)

// Primary indicates that a component should be given preference when
// multiple candidates are qualified to inject a single-valued dependency.
// If exactly one 'primary' component exists among the candidates, it
// will be the injected value.
//
// Example:
//
//	di.Register(func(repository FooRepository) FooService {
//		return &FooServiceImpl{ repository: repository }
//	})
//
//	di.Register(func() FooRepository {
//		return &MemoryRepositoryImpl{}
//	})
//
//	di.Register(func() FooRepository {
//		return &DatabaseRepositoryImpl{}
//	}, di.Primary)
//
// Because DatabaseRepositoryImpl is marked with Primary, it will be
// injected preferentially over the MemoryRepositoryImpl variant
// assuming both are present as component within the same di container.
func Primary(f *Factory) {
	_primaryQualifyConfig(f)
}
