package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// provider is a node in the dependency graph that represents a constructor provided by the user.
//
// constructorNodes can produce zero or more values that they storage into the container.
// For the Provide path, we verify that constructorNodes produce at least one value,
// otherwise the function will never be called.
type provider struct {
	order            int            // order of this node in graph
	scope            Scope          // provider scope
	isPrimary        bool           // defines a preference when multiple providers of the same type are present
	ctorValue        reflect.Value  // constructor function
	ctorType         reflect.Type   // type information about constructor
	params           []reflect.Type // type information about ctorValue parameters.
	useContext       bool           // first param is context.Context
	returnType       reflect.Type   // type information about ctorValue return.
	returnErrIdx     int            // error return index (-1, 1 or 2)
	returnCleanupIdx int            // cleanup function return index (-1, 1 or 2)
}

type callResult struct {
	err     error
	value   any
	cleanup reflect.Value
}

func (p *provider) call(ctx context.Context) (out *callResult, callError error) {

	if err := p.shallowCheckDependencies(); err != nil {
		callError = errors.Join(errors.New(fmt.Sprintf("missing dependencies for function %v", p.ctorType)), err)
		return
	}

	if args, pErr := p.getParams(ctx); pErr != nil {
		callError = pErr
	} else {
		out = &callResult{}
		results := p.ctorValue.Call(args)

		if p.returnErrIdx != -1 {
			cErr := results[p.returnErrIdx]
			if !cErr.IsNil() {
				out.err = cErr.Interface().(error)
			}
		}

		if p.returnCleanupIdx != -1 {
			cleanup := results[p.returnCleanupIdx]
			if !cleanup.IsNil() {
				out.cleanup = cleanup
			}
		}

		if out.err == nil {
			out.value = results[0].Interface()
		}
	}

	// results := c.invoker()(reflect.ValueOf(n.ctor), args)
	return
}

// BuildList returns an ordered list of values which may be passed directly
// to the underlying ctorValue.
func (p *provider) getParams(ctx context.Context) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(p.params))
	for i, param := range p.params {
		if p.useContext && i == 0 {
			args[0] = reflect.ValueOf(ctx)
			continue
		}
		v, err := resolve(ctx, param)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(v)
	}
	return args, nil
}

// Checks that all direct dependencies of the provided parameters are present in
// the container. Returns an error if not.
func (p *provider) shallowCheckDependencies() error {
	var err error

	missingDeps := p.findMissingDependencies()
	for _, dep := range missingDeps {
		err = errors.Join(err, errors.New(fmt.Sprintf("\t%v", dep)))
	}

	return err
}

func (p *provider) findMissingDependencies() []reflect.Type {
	var missingDeps []reflect.Type

	for i, param := range p.params {
		if p.useContext && i == 0 {
			// ignore context
			continue
		}
		allProviders := providers[param]
		// This means that there is no provider that provides this value,
		// and it is NOT being decorated and is NOT optional.
		// In the case that there is no providers but there is a decorated value
		// of this type, it can be provided safely so we can safely skip this.
		if len(allProviders) == 0 {
			missingDeps = append(missingDeps, param)
		}
	}
	return missingDeps
}

// createProvider
// func([context.Context], [dep A..N]) (A, [cleanup func()], [error])
func createProvider(ctor any) (*provider, error) {
	ctorType := reflect.TypeOf(ctor)
	if ctorType == nil {
		return nil, errors.New("can't createProvider an untyped nil")
	}
	if ctorType.Kind() != reflect.Func {
		return nil, errors.New(fmt.Sprintf("must createProvider constructor function, got %v (type %v)", ctor, ctorType))
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
			return nil, errors.New(fmt.Sprintf("context.Context should be the first parameter in fucntion %v", ctorType))
		}
		params = append(params, reflect.PointerTo(inType))
	}

	// results
	// func([context.Context], [dep A..N]) (ServiceA, [cleanup func()], [error])
	numOut := ctorType.NumOut()
	if numOut == 0 {
		return nil, errors.New(fmt.Sprintf("%v must createProvider one non-error type", ctorType))
	}

	returnType := ctorType.Out(0)
	returnErrIdx := -1
	returnCleanupIdx := -1

	if returnType.Kind() == reflect.Func || isError(returnType) {
		return nil, errors.New(fmt.Sprintf("%v has invalid return type (first) %v", ctorType, returnType))
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
				return nil, errors.New(fmt.Sprintf("%v has invalid cleanup signature (second) %v", ctorType, secondType))
			}
			hasCleanup = true
			returnCleanupIdx = 1
		} else if isError(secondType) {
			hasError = true
			returnErrIdx = 1
		} else {
			return nil, errors.New(fmt.Sprintf("%v has invalid return type (second) %v", ctorType, secondType))
		}

		if numOut == 3 {
			thirdType := ctorType.Out(2)
			if thirdType.Kind() == reflect.Func && !hasCleanup {
				if thirdType.NumIn() > 0 || thirdType.NumOut() > 0 {
					return nil, errors.New(fmt.Sprintf("%v has invalid cleanup signature (third) %v", ctorType, thirdType))
				}
				returnCleanupIdx = 2
			} else if isError(thirdType) && !hasError && hasCleanup {
				returnErrIdx = 2
			} else {
				return nil, errors.New(fmt.Sprintf("%v has invalid return type (third) %v", ctorType, thirdType))
			}
		}
		break
	default:
		return nil, errors.New(fmt.Sprintf("%v has invalid return", ctorType))
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

	//if _, ok := providers[callResult]; ok {
	//
	//}

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
		return nil, errors.New("this function introduces a cycle")
	}

	return p, nil
}

// Parameter 2 of constructor in yanbin.blog.testweb.controllers.ManagementController required a bean of type 'yanbin.blog.testweb.service.CalcEngineFactory' that could not be found
// org.springframework.beans.factory.NoSuchBeanDefinitionException: No qualifying bean of type [in.amruth.xplore.utility.IUtil] found for dependency: expected at least 1 bean which qualifies as autowire candidate for this dependency. Dependency annotations: {@org.springframework.beans.factory.annotation.Autowired(required=true)}
// Field dependency in com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanA required a bean of type 'com.baeldung.springbootmvc.nosuchbeandefinitionexception.BeanB' that could not be found.
// No qualifying bean of type
//  [com.baeldung.packageB.IBeanB] is defined:
//expected single matching bean but found 2: beanB1,beanB2
// https://www.baeldung.com/spring-nosuchbeandefinitionexception
