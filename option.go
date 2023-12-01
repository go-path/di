package di

var (
	configPrimary = configFn(func(p *provider) {
		p.isPrimary = true
	})
	configPrototype = configFn(func(p *provider) {
		p.scope = PrototypeScope
	})
	configContext = configFn(func(p *provider) {
		p.scope = ContextScope
	})
	configSingleton = configFn(func(p *provider) {
		p.scope = SingletonScope
	})
)

// ProviderOption is the type to replace default parameters.
// di.Provide accepts any number of options (this is functional option pattern).
type ProviderOption interface {
	apply(*provider)
}

func Primary() ProviderOption {
	return configPrimary
}

func Singleton() ProviderOption {
	return configSingleton
}

func Prototype() ProviderOption {
	return configPrototype
}

func Context() ProviderOption {
	return configContext
}

type configFn func(*provider)

func (f configFn) apply(p *provider) {
	f(p)
}
