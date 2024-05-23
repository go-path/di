package di

import (
	"context"
	"reflect"
)

// nilReturn internal representation of a daemon/service factory
// e. g. for a nil value returned from Factory
type nilReturn struct{}

type mockFunc func(ctx context.Context) any

type ConditionFunc func(Container, *Factory) bool

// Factory is a node in the dependency graph that represents a constructor provided by the user
// and the basic attributes of the returned component (if applicable)
type Factory struct {
	order          int                   // order of this node in graph
	key            reflect.Type          // key for this factory
	scope          string                // Factory scope
	startup        bool                  // will be initialized with container
	priority       int                   // priority of use
	factoryType    reflect.Type          // type information about constructor
	factoryValue   reflect.Value         // constructor function
	returnType     reflect.Type          // type information about return type.
	returnErrorIdx int                   // error return index (-1, 0 or 1)
	returnValueIdx int                   // value return index (0 or 1)
	parameters     []*Parameter          // information about factory parameters.
	parameterKeys  []reflect.Type        // type information about factory parameters.
	initializers   []Callback            // post construct callbacks
	disposers      []Callback            // disposal functions
	conditions     []ConditionFunc       // indicates that a component is only eligible for registration when all specified conditions match.
	qualifiers     map[reflect.Type]bool // component qualifiers
	mock           mockFunc
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

// Key gets the factory component Key (Key = reflect.TypeOf(ComponentType))
func (f *Factory) Key() reflect.Type {
	return f.key
}

// Scope gets the scope name
func (f *Factory) Scope() string {
	return f.scope
}

// Singleton returns true if the scope = 'singleton'
func (f *Factory) Singleton() bool {
	return f.scope == SCOPE_SINGLETON
}

// Prototype returns true if the scope = 'prototype'
func (f *Factory) Prototype() bool {
	return f.scope == SCOPE_PROTOTYPE
}

// Startup returns true if this factory is configured to run during the container initialization
func (f *Factory) Startup() bool {
	return f.startup
}

// Priority gets the priority of this components
func (f *Factory) Priority() int {
	return f.priority
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

// Disposers returns the list of disposer methods for this factory
func (f *Factory) Disposers() []Callback {
	return append([]Callback{}, f.disposers...)
}

// HasDisposers returns true if this factory has any disposer method
func (f *Factory) HasDisposers() bool {
	return len(f.disposers) > 0
}

// Conditions returns the list of conditions methods for this factory
// The component is only eligible for registration when all specified conditions match.
func (f *Factory) Conditions() []ConditionFunc {
	return append([]ConditionFunc{}, f.conditions...)
}

// HasConditions returns true if this factory has any condition method
func (f *Factory) HasConditions() bool {
	return len(f.conditions) > 0
}

// Primary returns true if this is a Primary candidate
// for a component (has the qualifier PrimaryQualifier)
func (f *Factory) Primary() bool {
	return f.HasQualifier(_primaryQualifierKey)
}

// Alternative returns true if this is a Alternative candidate
// for a component (has the qualifier AlternativeQualifier)
func (f *Factory) Alternative() bool {
	return f.HasQualifier(_alternativeQualifierKey)
}

// Qualifiers returns the list of qualifiers for this factory
func (f *Factory) Qualifiers() []reflect.Type {
	var o []reflect.Type
	for qualifier := range f.qualifiers {
		o = append(o, qualifier)
	}
	return o
}

// HasQualifier return true if this Factory has the specified qualifier
func (f *Factory) HasQualifier(q reflect.Type) bool {
	for qualifier := range f.qualifiers {
		if qualifier == q {
			return true
		}
	}
	return false
}

// Mock returns true if this is a Mock factory (testing)
func (f *Factory) Mock() bool {
	return f.mock != nil
}
