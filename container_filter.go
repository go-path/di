package di

import (
	"context"
	"reflect"
	"sort"
)

type FilteredFactories struct {
	container Container
	factories []*Factory
}

func (f *FilteredFactories) Sort(less func(a, b *Factory) bool) *FilteredFactories {
	if less == nil {
		less = DefaultFactorySortLessFn
	}

	sort.Slice(f.factories, func(i, j int) bool {
		return less(f.factories[i], f.factories[j])
	})

	return f
}

func (f *FilteredFactories) Foreach(visitor func(f *Factory) (stop bool, err error)) error {
	for _, p := range f.factories {
		if stop, err := visitor(p); err != nil {
			return err
		} else if stop {
			return nil
		}
	}
	return nil
}

func (f *FilteredFactories) Instances(ctx context.Context) ([]any, error) {
	var objects []any

	err := f.Foreach(func(factory *Factory) (bool, error) {
		if obj, _, err := f.container.GetObjectFactory(factory, true, ctx)(); err != nil {
			return true, err
		} else if obj != nil {
			objects = append(objects, obj)
		}
		return false, nil
	})

	return objects, err
}

func (c *container) Filter(options ...FactoryConfig) *FilteredFactories {

	filter := &Factory{
		qualifiers: make(map[reflect.Type]bool),
	}
	for _, option := range options {
		option(filter)
	}

	var factories []*Factory

	for _, candidates := range c.factories {
		for _, factory := range candidates {
			matchAll := true
			for _, match := range filter.conditions {
				if !match(c, factory) {
					matchAll = false
					break
				}
			}

			if !matchAll {
				continue
			}

			if filter.Primary() && !factory.Primary() {
				continue
			}

			if filter.startup && !factory.startup {
				continue
			}

			if filter.scope != "" && filter.scope != factory.scope {
				continue
			}

			if len(filter.qualifiers) > 0 {
				found := false
				for qualifier := range filter.qualifiers {
					if factory.qualifiers[qualifier] {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			factories = append(factories, factory)
		}
	}

	return (&FilteredFactories{
		container: c,
		factories: factories,
	}).Sort(DefaultFactorySortLessFn)
}

func DefaultFactorySortLessFn(a, b *Factory) bool {
	if a.Mock() != b.Mock() {
		// mock first (testing)
		return a.Mock()
	}

	if a.Primary() != b.Primary() {
		return a.Primary()
	}

	if a.Alternative() != b.Alternative() {
		return b.Alternative()
	}

	return a.Priority() < b.Priority()
}
