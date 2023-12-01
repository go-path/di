package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var pGraph = &graph{}

func MustProvide(ctor any, opts ...ProviderOption) {
	if err := Provide(ctor, opts...); err != nil {
		panic(err)
	}
}

// Provide
// func([context.Context], [dep A..N]) (A, [cleanup func()], [error])
func Provide(ctor any, opts ...ProviderOption) error {
	ctorType := reflect.TypeOf(ctor)
	if ctorType == nil {
		return errors.New("can't createProvider an untyped nil")
	}
	if ctorType.Kind() != reflect.Func {
		return errors.New(fmt.Sprintf("must createProvider constructor function, got %v (type %v)", ctor, ctorType))
	}

	// builds a params list from the provided constructor type.
	numArgs := ctorType.NumIn()
	if ctorType.IsVariadic() {
		numArgs--
	}

	params := make([]reflect.Type, 0, numArgs)

	for i := 0; i < numArgs; i++ {
		// @TODO: Qualifiers, like https://docs.oracle.com/javaee/6/api/javax/inject/Inject.html
		inType := ctorType.In(i)
		if isContext(inType) && i != 0 {
			return errors.New(fmt.Sprintf("context.Context should be the first parameter in fucntion %v", ctorType))
		}
		params = append(params, reflect.PointerTo(inType))
	}

	// results
	// func([context.Context], [dep A..N]) (ServiceA, [cleanup func()], [error])
	numOut := ctorType.NumOut()
	if numOut == 0 {
		return errors.New(fmt.Sprintf("%v must createProvider one non-error type", ctorType))
	}

	returnType := ctorType.Out(0)
	returnErrIdx := -1
	returnCleanupIdx := -1

	if returnType.Kind() == reflect.Func || isError(returnType) {
		return errors.New(fmt.Sprintf("%v has invalid return type (first) %v", ctorType, returnType))
	}

	switch numOut {
	case 1:
		break
	case 2:
	case 3:
		secondType := ctorType.Out(1)
		hasError := false
		hasCleanup := false
		if secondType.Kind() == reflect.Func {
			if secondType.NumIn() > 0 || secondType.NumOut() > 0 {
				return errors.New(fmt.Sprintf("%v has invalid cleanup signature (second) %v", ctorType, secondType))
			}
			hasCleanup = true
			returnCleanupIdx = 1
		} else if isError(secondType) {
			hasError = true
			returnErrIdx = 1
		} else {
			return errors.New(fmt.Sprintf("%v has invalid return type (second) %v", ctorType, secondType))
		}

		if numOut == 3 {
			thirdType := ctorType.Out(2)
			if thirdType.Kind() == reflect.Func && !hasCleanup {
				if thirdType.NumIn() > 0 || thirdType.NumOut() > 0 {
					return errors.New(fmt.Sprintf("%v has invalid cleanup signature (third) %v", ctorType, thirdType))
				}
				returnCleanupIdx = 2
			} else if isError(thirdType) && !hasError && hasCleanup {
				returnErrIdx = 2
			} else {
				return errors.New(fmt.Sprintf("%v has invalid return type (third) %v", ctorType, thirdType))
			}
		}
		break
	default:
		return errors.New(fmt.Sprintf("%v has invalid return", ctorType))
	}

	key := reflect.PointerTo(returnType)

	p := &provider{
		scope:            SingletonScope,
		ctorValue:        reflect.ValueOf(ctor),
		ctorType:         ctorType,
		params:           params,
		returnType:       returnType,
		returnErrIdx:     returnErrIdx,
		returnCleanupIdx: returnCleanupIdx,
	}
	p.order = pGraph.add(p)

	for _, opt := range opts {
		opt.apply(p)
	}

	// cache old providers before running cycle detection.
	oldProviders := providers[key]
	providers[key] = append(providers[key], p)

	if ok, cycle := pGraph.isAcyclic(); !ok {
		// When a cycle is detected, recover the old providers to reset
		// the providers map back to what it was before this node was
		// introduced.
		providers[key] = oldProviders
		fmt.Println(cycle)

		// , s.cycleDetectedError(cycle)
		return errors.New("this function introduces a cycle")
	}

	return nil
}

func MustGet[T any](contexts ...context.Context) T {
	o, err := Get[T](contexts...)
	if err != nil {
		panic(err)
	}
	return o
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

	if testingHasMock { // see Mock
		if fn, ok := testingMocks[key]; ok {
			o = fn(ctx)
			return
		}
	}

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
