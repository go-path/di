package di

import "reflect"

// Parameter representação de um parametro usado na injeção de dependencias
type Parameter struct {
	key          reflect.Type
	value        reflect.Type            // the value type
	provider     bool                    // is provider?  (Ex. func(sq Provider[*MyService])
	unmanaged    bool                    // is unmanaged provider?  (Ex. func(sq Unmanaged[*MyService])
	qualified    bool                    // is qualified?  (Ex. func(sq Qualified[*MyService, MyQualifier])
	qualifier    reflect.Type            // the qualifier type
	factories    map[*Factory]bool       // exactly matches the type
	candidates   map[*Factory]bool       // alternative matches (Ex. value = A, B implements A, B is candidate, if A is missing)
	funcWithImpl func(any) reflect.Value // used by Qualified and Provider
}

// Qualified indicates that this parameter is qualified (Ex. func(sq Qualified[*MyService, MyQualifier])
func (p *Parameter) Qualified() bool {
	return p.qualified
}

// Provider indicates that this parameter is a provider (Ex. func(sq Provider[*testService])
func (p *Parameter) Provider() bool {
	return p.provider
}

// Unmanaged indicates that this parameter is a unmanaged provider (Ex. func(sq Unmanaged[*testService])
func (p *Parameter) Unmanaged() bool {
	return p.unmanaged
}

func (p *Parameter) Key() reflect.Type {
	return p.key
}

func (p *Parameter) Value() reflect.Type {
	return p.value
}

func (p *Parameter) Qualifier() reflect.Type {
	return p.qualifier
}

// Factories list all candidates that exactly matches the type
func (p *Parameter) Factories() (list []*Factory) {
	for f := range p.factories {
		list = append(list, f)
	}
	return
}

// Candidates list alternative matches (Ex. value = A, B implements A, B is candidate, if A is missing)
func (p *Parameter) Candidates() (list []*Factory) {
	for f := range p.candidates {
		list = append(list, f)
	}
	return
}

// HasCandidates checks if there is any candidate for this parameter
func (p *Parameter) HasCandidates() bool {
	return len(p.factories) > 0 || len(p.candidates) > 0
}

func (p *Parameter) ValueOf(value any) reflect.Value {
	return p.funcWithImpl(value)
}

func (p *Parameter) IsValidCandidate(f *Factory) (isCandidate bool, isExactMatch bool) {

	if f.key == p.key {
		isCandidate = true
		isExactMatch = true
	} else if f.key.AssignableTo(p.key) {
		isCandidate = true
		isExactMatch = false
	}

	if !isCandidate {
		if p.Qualified() {
			if !f.HasQualifier(p.qualifier) {
				isCandidate = false
				isExactMatch = false
			} else if f.key == p.value {
				isCandidate = true
				isExactMatch = true
			} else if f.key.AssignableTo(p.value) {
				isCandidate = true
				isExactMatch = false
			}
		} else if p.Provider() {
			if f.key == p.value {
				isCandidate = true
				isExactMatch = true
			} else if f.key.AssignableTo(p.value) {
				isCandidate = true
				isExactMatch = false
			}
		}
	}

	return
}
