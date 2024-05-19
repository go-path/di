package di

import (
	"reflect"
	"strings"
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
	value T
}

func (q Qualified[T, Q]) Get() T {
	return q.value
}

func (q Qualified[T, Q]) Type() reflect.Type {
	return Key[T]()
}

func (q Qualified[T, Q]) Qualifier() reflect.Type {
	return Key[Q]()
}

// Qualify register a qualifier for the component
//
// Example:
//
//	type MyQualifier string
//
//	di.Register(func() *MyService {
//		return &MyService{}
//	}, di.Qualify[MyQualifier]())
func Qualify[Q any]() ComponentConfig {
	qualifier := Key[Q]()
	return func(f *Factory) {
		f.qualifiers[qualifier] = true
	}
}

type MyQualifier string

func init() {

	// produzir um componente qualificado
	global.Register(
		func() *Factory {
			return &Factory{}
		},
		Primary,
		Qualify[MyQualifier](),
	)

	// usar uma dependencia que possui um qualificador
	global.Register(func(f Qualified[*Factory, MyQualifier]) {
		factory := f.Get()
		print(factory.Primary())
	})

	// ConvertibleTo, Implements, AssignableTo

	factoryType := reflect.TypeOf(xxx)
	numParams := factoryType.NumIn()
	for i := 0; i < numParams; i++ {
		tin := factoryType.In(i)
		if tin.Kind() == reflect.Struct && strings.HasPrefix(tin.String(), "di.Qualified[") {
			var isValid bool
			var getValue reflect.Method
			var getQualifier reflect.Method

			getValue, isValid = tin.MethodByName("Type")
			if !isValid || getValue.Type.NumIn() != 1 || getValue.Type.NumOut() != 1 || !getValue.Type.Out(0).Implements(_typeReflectType) {
				isValid = false
			}

			if isValid {
				getQualifier, isValid = tin.MethodByName("Qualifier")
				if !isValid || getQualifier.Type.NumIn() != 1 || getQualifier.Type.NumOut() != 1 || !getQualifier.Type.Out(0).Implements(_typeReflectType) {
					isValid = false
				}
			}

			if isValid {
				nptr_vl := reflect.New(tin).Elem() // Value {di.Qualified[A, B]}

				args := []reflect.Value{nptr_vl}

				valueTypeResult := getValue.Func.Call(args)
				valueType := valueTypeResult[0].Interface().(reflect.Type)

				qualifierResult := getQualifier.Func.Call(args)
				qualifierType := qualifierResult[0].Interface().(reflect.Type)

				print(valueType.String())
				print(qualifierType.String())
			}
		}
	}
}

func xxx(f Qualified[*Factory, MyQualifier]) {

}
