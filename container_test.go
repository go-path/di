package di

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type testQualifier string

type testService interface {
	Exec()
	Name() string
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
	name     string
	callback func(s testService, event string)
}

func (s *testServiceImpl) Name() string {
	return s.name
}

func (s *testServiceImpl) Exec() {
	if s.callback != nil {
		s.callback(s, "exec")
	}
}

func (s *testServiceImpl) Initialize() {
	if s.callback != nil {
		s.callback(s, "Initialize")
	}
}

func (s *testServiceImpl) Destroy() {
	if s.callback != nil {
		s.callback(s, "Destroy")
	}
}

var (
	t0 = reflect.TypeOf((*testService)(nil))  // 0
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
	t.Run("func constructor", func(t *testing.T) {

		ctn := New(nil)

		var (
			err     error
			called  bool
			result  testService
			service = &testServiceImpl{}
		)

		require.NoError(t, ctn.ShouldRegister(func() testService {
			require.False(t, called, "constructor must be called exactly once")
			called = true
			return service
		}))

		require.NoError(t, ctn.Initialize())

		result, err = Get[testService](ctn)

		require.NoError(t, err)
		require.True(t, called, "constructor must be called first")
		require.NotNil(t, result, "invoke got nil service")
		require.Equal(t, service, result, "service must match constructor's return value")
	})
}

func TestFullCycle(t *testing.T) {
	t.Run("func constructor", func(t *testing.T) {

		ctn := New(nil)

		var events []string

		appendEvent := func(s testService, event string) {
			events = append(events, s.Name()+":"+event)
		}

		count := 0

		// qualified component
		ctn.Register(
			func() testService {
				count++
				return &testServiceImpl{
					name:     "srv-" + strconv.Itoa(count),
					callback: appendEvent,
				}
			},
			Primary,
			Qualify[testQualifier](),
			Initializer(func(s testService) {
				appendEvent(s, "Initializer()")
			}),
			Disposer(func(s testService) {
				appendEvent(s, "Disposer()")
			}),
		)

		// Qualified
		require.NoError(t, ctn.ShouldRegister(func(q Qualified[testService, testQualifier]) {
			s := q.Get()
			appendEvent(s, "Qualified") // expect "srv-1"
		}, Startup(100)))

		// Provider (same instance)
		require.NoError(t, ctn.ShouldRegister(func(q Provider[testService]) {
			s, _ := q.Get()
			appendEvent(s, "Provided") // expect "srv-1"

			s, _ = q.Get()
			appendEvent(s, "Provided") // expect "srv-1"
		}, Startup(200)))

		// Unmanaged (returns new instance)
		require.NoError(t, ctn.ShouldRegister(func(u Unmanaged[testService]) {
			sa, da, _ := u.Get()
			appendEvent(sa, "Unmanaged") // expect "srv-2"

			sb, db, _ := u.Get()
			appendEvent(sb, "Unmanaged") // expect "srv-3"

			da.Dispose()
			db.Dispose()
		}, Startup(300)))

		require.NoError(t, ctn.Initialize())

		require.Equal(t, []string{
			"srv-1:Initializer()", "srv-1:Qualified", "srv-1:Provided", "srv-1:Provided",
			"srv-2:Initializer()", "srv-2:Unmanaged",
			"srv-3:Initializer()", "srv-3:Unmanaged",
			"srv-2:Disposer()",
			"srv-3:Disposer()",
		}, events)
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
