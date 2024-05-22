package di

import (
	"context"
	"reflect"
)

// nilReturn internal representation of a nil component instance, e. g. for a nil value returned from Factory
type nilReturn struct{}

type mockFn func(ctx context.Context) any

type ConditionFn func(Container, *Factory) bool

// Factory is a node in the dependency graph that represents a constructor provided by the user
// and the basic attributes of the returned component (if applicable)
type Factory struct {
	order          int                   // order of this node in graph
	key            reflect.Type          // key for this factory
	factoryType    reflect.Type          // type information about constructor
	factoryValue   reflect.Value         // constructor function
	returnType     reflect.Type          // type information about return type.
	returnErrorIdx int                   // error return index (-1, 0 or 1)
	returnValueIdx int                   // value return index (0 or 1)
	parameters     []*Parameter          // information about factory parameters.
	parameterKeys  []reflect.Type        // type information about factory parameters.
	scope          string                // Factory scope
	priority       int                   // priority of use
	startup        bool                  // will be initialized with container
	initializers   []reflect.Value       // post construct callbacks
	disposers      []reflect.Value       // disposal functions
	conditions     []ConditionFn         // indicates that a component is only eligible for registration when all specified conditions match.
	qualifiers     map[reflect.Type]bool // component qualifiers
	mock           mockFn
}

// Create a new instance of component.
func (f *Factory) Create(args []reflect.Value) (any, error) {

	results := f.factoryValue.Call(args)

	if f.ReturnsError() {
		if err := results[f.returnErrorIdx]; !err.IsNil() {
			return nil, err.Interface().(error)
		}
	}

	if !f.ReturnsValue() {
		return nil, nil
	}

	return results[f.returnValueIdx].Interface(), nil
}

func (f *Factory) Singleton() bool {
	return f.scope == SCOPE_SINGLETON
}

func (f *Factory) Scope() string {
	return f.scope
}

func (f *Factory) Primary() bool {
	return f.HasQualifier(_primaryQualifierKey)
}

func (f *Factory) Alternative() bool {
	return f.HasQualifier(_alternativeQualifierKey)
}

func (f *Factory) Mock() bool {
	return f.mock != nil
}

func (f *Factory) Constructor() reflect.Value {
	return f.factoryValue
}

func (f *Factory) Params() []reflect.Type {
	return f.parameterKeys[0:]
}

func (f *Factory) Type() reflect.Type {
	return f.returnType
}

func (f *Factory) ReturnsError() bool {
	return f.returnErrorIdx != -1
}

func (f *Factory) ReturnsValue() bool {
	return f.returnValueIdx != -1
}

func (f *Factory) ReturnErrorIdx() int {
	return f.returnErrorIdx
}

func (f *Factory) ReturnValueIdx() int {
	return f.returnValueIdx
}

func (f *Factory) Disposer() bool {
	return len(f.disposers) > 0
}

func (f *Factory) Startup() bool {
	return f.startup
}

func (f *Factory) Priority() int {
	return f.priority
}

func (f *Factory) HasQualifier(q reflect.Type) bool {
	for qualifier := range f.qualifiers {
		if qualifier == q {
			return true
		}
	}
	return false
}
