package di

import (
	"context"
	"sync"
)

func newSingletonScope() *scopeSingleton {
	return &scopeSingleton{
		objects:   make(map[int]any),
		disposers: make(map[int]DisposableAdapter),
	}
}

// todo sync.Map?
type scopeSingleton struct {
	m         sync.RWMutex
	objects   map[int]any               // Cache of singleton objects
	disposers map[int]DisposableAdapter // Cache of disposers
}

func (s *scopeSingleton) Get(ctx context.Context, factory *Factory, createObject CreateObjectFunc) (any, error) {
	fid := factory.Id()
	if singleton, exist := s.objects[fid]; exist {
		return singleton, nil
	}

	if singleton, disposer, err := createObject(); err != nil {
		return nil, err
	} else {

		s.m.Lock()
		if singleton, exist := s.objects[fid]; exist {
			s.m.Unlock()
			if disposer != nil {
				disposer.Dispose()
			}
			return singleton, nil
		}

		defer s.m.Unlock()

		if singleton != nil {
			s.objects[fid] = singleton
		}

		if disposer != nil {
			s.disposers[fid] = disposer
		}

		return singleton, nil
	}
}

func (s *scopeSingleton) Remove(*Factory, any) (any, error) {
	return nil, nil
}

func (s *scopeSingleton) Destroy() {
	s.m.Lock()
	disposers := s.disposers
	s.disposers = make(map[int]DisposableAdapter)
	s.objects = make(map[int]any)
	defer s.m.Unlock()
	for _, disposer := range disposers {
		disposer.Dispose()
	}
}

// getSingleton Return the (raw) singleton object registered under the given key.
func (s *scopeSingleton) getSingleton(fid int) any {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.objects[fid]
}
