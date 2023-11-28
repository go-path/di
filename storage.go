package di

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
)

type ctxKeyType int // unexported type for ctxKey to avoid collisions.

var (
	ctxKey           ctxKeyType
	ctxSeq           atomic.Uint64
	providers        = map[reflect.Type][]*provider{}
	contextStorages  = sync.Map{}
	singletonStorage = &storage{}
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
			if result.cleanup.IsValid() {
				result.cleanup.Call(nil)
			}
		}
		s.values.Delete(key)
		return true
	})
}

func getStore(ctx context.Context) (*storage, bool) {
	if id, ok := ctx.Value(ctxKey).(uint64); !ok {
		return nil, false
	} else if s, sok := contextStorages.Load(id); !sok {
		return nil, false
	} else {
		return s.(*storage), true
	}
}

func Cleanup() {
	contextStorages.Range(func(key, value any) bool {
		contextStorages.Delete(key)
		if s, ok := value.(*storage); ok {
			s.cleanup()
		}
		return true
	})
	singletonStorage.cleanup()
}

func NewContext(ctx context.Context) context.Context {
	id := ctxSeq.Add(1)
	ctx = context.WithValue(ctx, ctxKey, id)

	s := &storage{}
	contextStorages.Store(id, s)

	go func() {
		<-ctx.Done()
		contextStorages.Delete(id)
		s.cleanup()
		s = nil
	}()

	return ctx
}
