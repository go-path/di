package di

import "sort"

type FilteredFactories struct {
	container Container
	factories []*Factory
}

func (f *FilteredFactories) Sort(less func(a, b *Factory) bool) *FilteredFactories {
	if less == nil {
		less = FactorySortLessFn
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

func (c *container) Filter(options ...ComponentConfig) *FilteredFactories {

	filter := &Factory{}
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

			if filter.primary && !factory.primary {
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

	return &FilteredFactories{
		container: c,
		factories: factories,
	}
}

func FactorySortLessFn(a, b *Factory) bool {
	if a.Primary() {
		return true
	} else if b.Primary() {
		return false
	}

	if a.Mock() != b.Mock() {
		return b.Mock()
	}

	if a.Startup() && b.Startup() {
		return a.StartupPriority() < b.StartupPriority()
	}

	return a.Priority() < b.Priority()
}
