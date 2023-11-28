package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var pGraph = &graph{}

func Primary(ctor any, scope ...Scope) error {
	p, err := createProvider(ctor)
	if err == nil {
		if len(scope) > 0 {
			p.scope = scope[0]
		}
		p.isPrimary = true
	}
	return err
}

func Singleton(ctor any) error {
	_, err := createProvider(ctor) // default scope
	return err
}

func Context(ctor any) error {
	p, err := createProvider(ctor)
	if err == nil {
		p.scope = ContextScope
	}
	return err
}

func Prototype(ctor any) error {
	p, err := createProvider(ctor)
	if err == nil {
		p.scope = PrototypeScope
	}
	return err
}

func Get[T any](contexts ...context.Context) (o T, e error) {
	var ctx context.Context
	if len(contexts) > 0 {
		ctx = contexts[0]
	} else {
		ctx = context.Background()
	}
	var instance *T
	key := reflect.TypeOf(instance)

	if v, err := resolve(ctx, key); err != nil {
		e = err
	} else {
		o = v.(T)
	}
	return
}

func resolve(ctx context.Context, key reflect.Type) (o any, e error) {
	//var instance *T
	//key := reflect.TypeOf(instance)

	// get candidates
	candidates := providers[key]

	var p *provider

	switch len(candidates) {
	case 0:
		e = errors.New(fmt.Sprintf("%v não foi encontrado candidato para o tipo", key))
	case 1:
		p = candidates[0]
	default:
		// check for primary
		for _, candidate := range candidates {
			if candidate.isPrimary {
				p = candidate
				break
			}
		}
		if p == nil {
			e = errors.New(fmt.Sprintf("%v multiplos candidatos para o tipo", key))
		}
	}
	if e != nil {
		return
	}

	var s *storage

	if p.scope == SingletonScope {
		s = singletonStorage
	} else if p.scope == ContextScope {
		if ss, ok := getStore(ctx); !ok {
			e = errors.New(fmt.Sprintf("o provider %v requer a inicialização do contexto", p))
			return
		} else {
			s = ss
		}
	}

	if s != nil {
		// first check if the storage already has cached a value for the type.
		if result, ok := s.get(key); ok {
			o = result.value
			return
		}
	}

	// callResult provider
	if result, errCall := p.call(ctx); errCall != nil {
		e = errCall
	} else {
		// return
		e = result.err
		if e == nil && result.value != nil {
			o = result.value
			if s != nil {
				s.set(key, result)
			}
		}
	}
	return
}
