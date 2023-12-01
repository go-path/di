package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

var _mockMu = sync.Mutex{}

// Mock allows mocking of a dependency. Accepts "T" or "func([context.Context]) T"
func Mock[T any](mock any) (cleanup func()) {
	if !testing.Testing() {
		panic("mocks are only allowed during testing")
	}

	var fn mockFn

	if t, idT := mock.(T); idT {
		fn = func(ctx context.Context) any {
			return t
		}
	} else if tFn, isFn := mock.(func() T); isFn {
		fn = func(ctx context.Context) any {
			return tFn()
		}
	} else if tFnCtx, isFnCtx := mock.(func(ctx context.Context) T); isFnCtx {
		fn = func(ctx context.Context) any {
			return tFnCtx(ctx)
		}
	} else {
		panic(fmt.Sprintf("mock must be 'T' or 'func([context.Context]) T', got %T", mock))
	}

	key := reflect.TypeOf((*T)(nil))

	_mockMu.Lock()
	defer _mockMu.Unlock()

	testingHasMock = true
	testingMocks[key] = fn

	// cleanup
	cleanup = func() {
		_mockMu.Lock()
		defer _mockMu.Unlock()

		delete(testingMocks, key)
		testingHasMock = len(testingMocks) > 0
	}
	return
}
