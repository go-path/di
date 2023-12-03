package di

type defaultQualifier string

type Qualifier[T any, Q any] interface {
	Value() T
}

func Qualify[T any, Q any](value T) Qualifier[T, Q] {
	return &qualified[T, Q]{value: value}
}

type qualified[T any, Q any] struct {
	value T
}

func (q *qualified[T, Q]) Value() T {
	return q.value
}
