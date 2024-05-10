package di

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
)

type ctxKeyType int // unexported type for ctxKey to avoid collisions.
type mockFn func(ctx context.Context) any

var (
	ctxKey    ctxKeyType
	ctxSeq    atomic.Uint64
	providers = map[reflect.Type][]*Provider{}
)

type storage struct {
	values sync.Map
}

func (s *storage) get(key reflect.Type) (*callResult, bool) {
	if result, ok := s.values.Load(key); !ok {
		return nil, false
	} else {
		return result.(*callResult), true
	}
}

func (s *storage) set(key reflect.Type, value *callResult) {
	s.values.Store(key, value)
}

func (s *storage) cleanup() {
	s.values.Range(func(key, value any) bool {
		if result, ok := value.(*callResult); ok {
			if result.isDisposer && result.value != nil {
				if d, isD := result.value.(Disposer); isD {
					d.Disposes()
				}
			}
		}
		s.values.Delete(key)
		return true
	})
}
