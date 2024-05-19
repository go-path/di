package di

import (
	"context"
	"reflect"
	"sync"
)

func newSingletonScope() *scopeSingleton {
	return &scopeSingleton{
		objects: make(map[reflect.Type]any),
	}
}

// todo sync.Map?
type scopeSingleton struct {
	m         sync.RWMutex
	objects   map[reflect.Type]any               // Cache of singleton objects
	disposers map[reflect.Type]DisposableAdapter // Cache of disposers
}

func (s *scopeSingleton) Get(ctx context.Context, key reflect.Type, factory ObjectFactory) (any, error) {
	if singleton := s.getSingleton(key); singleton != nil {
		return singleton, nil
	}

	// if singleton, exist := s.objects[key]; exist {
	// 	return singleton, nil
	// }

	if singleton, disposer, err := factory(); err != nil {
		return nil, err
	} else {

		s.m.Lock()
		if singleton, exist := s.objects[key]; exist {
			s.m.Unlock()
			if disposer != nil {
				disposer.Dispose()
			}
			return singleton, nil
		}

		defer s.m.Unlock()

		if singleton != nil {
			s.objects[key] = singleton
		}

		if disposer != nil {
			s.disposers[key] = disposer
		}

		return singleton, nil
	}
}

func (s *scopeSingleton) Remove(reflect.Type, any) (any, error) {
	return nil, nil
}

func (s *scopeSingleton) Destroy() {

}

// getSingleton Return the (raw) singleton object registered under the given key.
func (s *scopeSingleton) getSingleton(key reflect.Type) any {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.objects[key]
}
