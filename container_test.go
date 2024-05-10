package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type testService0 interface {
	Exec()
}
type testService1 interface {
	Exec()
}

type testService2 interface {
	Exec()
}

type testService3 interface {
	Exec()
}

type testService4 interface {
	Exec()
}

type testService5 interface {
	Exec()
}

type testServiceImpl struct {
	callback func()
}

func (i *testServiceImpl) Exec() {
	if i.callback != nil {
		i.callback()
	}
}

var (
	t0 = reflect.TypeOf((*testService0)(nil)) // 0
	t1 = reflect.TypeOf((*testService1)(nil)) // 1
	t2 = reflect.TypeOf((*testService2)(nil)) // 2
	t3 = reflect.TypeOf((*testService3)(nil)) // 3
	t4 = reflect.TypeOf((*testService4)(nil)) // 4
	t5 = reflect.TypeOf((*testService5)(nil)) // 5
)

type testProvider struct {
	key    reflect.Type
	params []reflect.Type
}

func testParams(types ...reflect.Type) []reflect.Type {
	params := make([]reflect.Type, 0, len(types))
	params = append(params, types...)
	return params
}

func TestEndToEndSuccess(t *testing.T) {
	Clear()

	t.Run("func constructor", func(t *testing.T) {

		var (
			err     error
			called  bool
			result  testService0
			service = &testServiceImpl{}
		)
		err = Provide(func() testService0 {
			require.False(t, called, "constructor must be called exactly once")
			called = true
			return service
		})
		require.NoError(t, err)

		result, err = Get[testService0]()

		require.NoError(t, err)
		require.True(t, called, "constructor must be called first")
		require.NotNil(t, result, "invoke got nil service")
		require.Equal(t, service, result, "service must match constructor's return value")
	})
}

//type aServiceImpl struct {
//}
//
//func (a *aServiceImpl) doSomething() {
//
//}

//func TestRegister(t *testing.T) {
//
//	var GetServiceA func() testService0
//
//	Register()
//	GetServiceA = Register(func() testService0 {
//		return &aServiceImpl{}
//	})
//}
