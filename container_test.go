package di

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type testQualifierA uint8
type testQualifierB uint8

type testServiceBase interface {
	Name() string
	Event(event string)
}

type testServiceA interface {
	testServiceBase
	IsA()
}

type testServiceB interface {
	testServiceBase
	IsB()
}

type testLoggger func(name, event string)

func newTestLogger() (logger testLoggger, logs func() []string) {
	var entries []string

	logger = func(name, event string) {
		entries = append(entries, name+":"+event)
	}

	logs = func() []string {
		return entries
	}

	return
}

type testServiceBaseImp struct {
	name   string
	logger testLoggger
}

func (s *testServiceBaseImp) Name() string { return s.name }

func (s *testServiceBaseImp) Event(event string) {
	if s.logger != nil {
		s.logger(s.name, event)
	}
}

func (s *testServiceBaseImp) Initialize() {
	s.Event("Initialize")
}

func (s *testServiceBaseImp) Destroy() {
	s.Event("Destroy")
}

type testServiceAImpl struct{ *testServiceBaseImp }

func (s *testServiceAImpl) IsA() {}

type testServiceBImpl struct{ *testServiceBaseImp }

func (s *testServiceBImpl) IsB() {}

func newTestServiceA(name string, logger testLoggger) testServiceA {
	return &testServiceAImpl{&testServiceBaseImp{name: name, logger: logger}}
}

func newTestServiceB(name string, logger testLoggger) testServiceB {
	return &testServiceBImpl{&testServiceBaseImp{name: name, logger: logger}}
}

func TestComponentKey(t *testing.T) {

	ctn := New(nil)
	logger, logs := newTestLogger()

	// Component Type = Key = typeof testServiceA
	ctn.Register(func() testServiceA {
		return newTestServiceA("a", logger)
	})

	// Component Type = Key = typeof *testServiceBImpl
	ctn.Register(func() *testServiceBImpl {
		return newTestServiceB("b", logger).(*testServiceBImpl)
	})

	// a Key = typeof testServiceA, exact match
	// b Key = typeof testServiceB, assignable match
	// c Key = typeof *testServiceBImpl, exact match
	ctn.Register(func(a testServiceA, b testServiceB, c *testServiceBImpl) {
		require.True(t, b == c, "b testServiceB != d *testServiceBImpl")
	}, Startup(100))

	// missing dependencies
	// func(di.testServiceA, di.testServiceB, *di.testServiceAImpl, *di.testServiceBImpl) depends on missing dependency
	// missing dependencies: *di.testServiceAImpl

	require.NoError(t, ctn.Initialize())

	require.Equal(t, []string{
		"a:Initialize",
		"b:Initialize",
	}, logs())
}

func TestConstructor(t *testing.T) {
	ctn := New(nil)

	var (
		err     error
		called  bool
		result  testServiceA
		service = newTestServiceA("test", nil)
	)

	require.NoError(t, ctn.ShouldRegister(func() testServiceA {
		require.False(t, called, "constructor must be called exactly once")
		called = true
		return service
	}))

	require.NoError(t, ctn.Initialize())

	result, err = GetFrom[testServiceA](ctn)

	require.NoError(t, err)
	require.True(t, called, "constructor must be called first")
	require.NotNil(t, result, "invoke got nil service")
	require.Equal(t, service, result, "service must match constructor's return value")
}

func TestFullCycle(t *testing.T) {
	ctn := New(nil)
	logger, logs := newTestLogger()

	count := 0

	// qualified component
	ctn.Register(
		func() testServiceA {
			count++
			return newTestServiceA("srv-"+strconv.Itoa(count), logger)
		},
		Primary,
		Qualify[testQualifierA](),
		Initializer(func(s testServiceA) {
			s.Event("Initializer()")
		}),
		Disposer(func(s testServiceA) {
			s.Event("Disposer()")
		}),
	)

	// Qualified
	require.NoError(t, ctn.ShouldRegister(func(q Qualified[testServiceA, testQualifierA]) {
		s := q.Get()
		s.Event("Qualified") // expect "srv-1"
	}, Startup(100)))

	// Provider (same instance)
	require.NoError(t, ctn.ShouldRegister(func(q Provider[testServiceA]) {
		s, _ := q.Get()
		s.Event("Provided") // expect "srv-1"

		s, _ = q.Get()
		s.Event("Provided") // expect "srv-1"
	}, Startup(200)))

	// Unmanaged (returns new instance)
	require.NoError(t, ctn.ShouldRegister(func(u Unmanaged[testServiceA]) {
		sa, da, _ := u.Get()
		sa.Event("Unmanaged") // expect "srv-2"

		sb, db, _ := u.Get()
		sb.Event("Unmanaged") // expect "srv-3"

		da.Dispose()
		db.Dispose()
	}, Startup(300)))

	require.NoError(t, ctn.Initialize())

	require.Equal(t, []string{
		"srv-1:Initialize", "srv-1:Initializer()", "srv-1:Qualified", "srv-1:Provided", "srv-1:Provided",
		"srv-2:Initialize", "srv-2:Initializer()", "srv-2:Unmanaged",
		"srv-3:Initialize", "srv-3:Initializer()", "srv-3:Unmanaged",
		"srv-2:Destroy", "srv-2:Disposer()",
		"srv-3:Destroy", "srv-3:Disposer()",
	}, logs())
}
