package di

import (
	"reflect"
)

type defaultQualifier string

type Qualifier[T any, Q any] interface {
	Value() T
	getValueType() reflect.Type
	getQualifierType() reflect.Type
}

func Qualify[T any, Q any](value T) Qualifier[T, Q] {
	var q *Q
	return &qualified[T, Q]{
		value:         value,
		valueType:     reflect.TypeOf(value),
		qualifierType: reflect.TypeOf(q),
	}
}

type qualified[T any, Q any] struct {
	value         T
	valueType     reflect.Type
	qualifierType reflect.Type
}

func (q *qualified[T, Q]) Value() T {
	return q.value
}

func (q *qualified[T, Q]) getValueType() reflect.Type {
	return q.valueType
}

func (q *qualified[T, Q]) getQualifierType() reflect.Type {
	return q.qualifierType
}
