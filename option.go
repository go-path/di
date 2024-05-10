package di

// ProviderOption is the type to replace default parameters.
// di.Provide accepts any number of options (this is functional option pattern).
type ProviderOption interface {
	apply(*Provider)
}

type configFn func(*Provider)

func (f configFn) apply(p *Provider) {
	f(p)
}

var (
	configPrimary = configFn(func(p *Provider) {
		p.isPrimary = true
	})
	configPrototype = configFn(func(p *Provider) {
		p.scope = PrototypeScope
	})
	configContext = configFn(func(p *Provider) {
		p.scope = ContextScope
	})
	configSingleton = configFn(func(p *Provider) {
		p.scope = SingletonScope
	})
)

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

// Startup indicates that this component must be initialized
// during container initialization
func Startup(priority int) ProviderOption {
	return configFn(func(p *Provider) {
		p.isStartup = true
		p.startupPriority = priority
	})
}
