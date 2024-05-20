package di

// Unmanaged Provider allow the creation of unmanaged instances. The container
// will not keep track of these instances and the application will be responsible
// for cleaning them up.
//
// Example:
//
//	di.Register(func(up di.Unmanaged[*Seat]) {
//		driver, err := sp.Get()
//		passenger, err := sp.Get()
//	})
//
// See Provider
type Unmanaged[T any] struct {
	TypeBase[T]
	supplier func() (any, DisposableAdapter, error)
}

// Get the value
func (u Unmanaged[T]) Get() (o T, a DisposableAdapter, e error) {
	if v, d, err := u.supplier(); err != nil {
		e = err
	} else {
		a = d
		o = v.(T)
	}
	return
}

func (u Unmanaged[T]) With(supplier func() (any, DisposableAdapter, error)) Unmanaged[T] {
	return Unmanaged[T]{TypeBase: TypeBase[T]{}, supplier: supplier}
}

func (u Unmanaged[T]) Unmanaged() bool {
	return true
}
