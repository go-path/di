# di [WIP]

This is The Way to do Dependency injection in Go.

```go

import "github.com/go-path/di"

type JediService interface {
    FeelTheForce()
}

type PadawanController struct {
    s JediService
}

func (p * PadawanController) Initialize() {
    p.s.FeelTheForce()
}

// register as startup component, injecting dependencies
di.Register(func(s JediService) *PadawanController {
	return &PadawanController{s:s}
}, di.Startup(100))


// (...) in a package far, far away ...

type yodaServiceImp struct {}

func (s * yodaServiceImp) FeelTheForce() {
    print("Patience you must have my young Padawan")
}

di.Register(&yodaServiceImp{})

// ... and
di.Initialize() 
```

This implementation has the following main objectives:

- Easy to understand
- Typesafe, using generics
- Extensible and configurable
- A foundation for the development of more complex frameworks and architectural structures.

---
> ARE YOU ALLERGIC? ATTENTION!
> 
> Some concepts applied in this library may contain gluten and/or be inspired by the implementations of [CDI Java](https://www.cdi-spec.org/) and [Spring IoC](https://docs.spring.io/spring-framework/reference/core/beans.html).
> 
> And obviously, this implementation uses [reflection](https://pkg.go.dev/reflect), which for some uninformed individuals can be detrimental. It is important to mention that we thoroughly sanitize our hands before handling any `Type` or `Value`.
>
> We will not be held responsible if you become a productive developer after coming into contact with any part of this library.
>
> "Your path you must decide.” — Yoda
---


**Don't know what DI is?** [Wikipedia](https://en.wikipedia.org/wiki/Dependency_injection)

The diagram below provides a summarized overview of how this DI container functions.

* **Container**: The current implementation allows the developer to register and obtain their components.
* **Component**: Any object (simple or complex) that the developer wishes to register in the container to be used by other (injected) components. During dependency injection, the injected constructor does not know how the requested component will be created. The container resolves the dependency, simplifying the development of complex systems.
* **Factory**: It informs how an instance of a component can be created. The developer registers a component constructor or even an instance in the container, and the container performs the necessary processing to generate the factory for that component and resolve its dependencies when the constructor is invoked. It is possible to have more than one factory for the same type of component. The container ensures that the correct instance is obtained according to the rules described later in this document.
* **Scope**: It manages the lifecycle of an instance. The developer specifies the scope of the component during registration. When the component is requested, the container asks the defined scope for the instance. The scope can return an existing instance or create a new one. This allows for the instantiation of components with various lifecycles; for example, instances that will be discarded at the end of a web request and other singleton instances that will remain alive until the entire application is terminated.

![Diagram 1](docs/diagram-1.png)

## Docs

### Installation

`go get github.com/go-path/di` 

### Container

[`di.Container`](https://github.com/go-path/di/blob/main/container.go#L17) is owr IoC Container.

If necessary, you can instantiate new containers using the method `New(parent Container) Container`. We've already registered a [global](https://github.com/go-path/di/blob/main/global.go) container and exposed all methods to simplify the library's usage.

Usually, you only need to interact with the method `di.Register(ctor any, opts ...FactoryConfig)` for component registration and finally the method `di.Initialize(contexts ...context.Context) error` for the container to initialize the components configured as 'Startup'.

When conducting unit tests in your project, take a look at method `Mock(mock any) (cleanup func())`.

If you're building a more complex architecture in your organization or a library with DI support, you'll likely use the methods below and others documented in the API to have full control over the components.

```
RegisterScope(name string, scope ScopeI) error

Get(key reflect.Type, ctx ...context.Context) (any, error)
Filter(options ...FactoryConfig) *FilteredFactories

Contains(key reflect.Type) bool

GetObjectFactory(factory *Factory, managed bool, ctx ...context.Context) CreateObjectFunc
GetObjectFactoryFor(key reflect.Type, managed bool, ctx ...context.Context) CreateObjectFunc
ResolveArgs(factory *Factory, ctx ...context.Context) ([]reflect.Value, error)
Destroy() error

DestroyObject(key reflect.Type, object any) error
DestroySingletons() error
```

## Registrar Componentes e Serviços/Daemons

Nós registramos serviços usando o método `di.Register(ctor any, opts ...FactoryConfig)`. NÃO ENTRE EM PÂNICO, mas esse método causa `panic` se houver algum erro no registro do componente. Isso acontece pois o esperado é que o registro de um componente não falhe. Se você precisar gerenciar o registro de um componente, utilize o método alternativo `ShouldRegister(ctor any, opts ...FactoryConfig) error` que retorna um erro.

O formato geral de registro de um componente é:

![di.Register utilization](docs/register.png)

```go
di.Register(
    func(a DependencyA, b DependencyB) MyComponent, error {
        return &myComponentImpl{a:a, b:b}
    }, 
    di.Primary, di.Priority(1)
)
```

Caso o componente não precise de um construtor (true Singleton), utilize o padrão abaixo, retornando a instancia que será usada em todo o container.

```go
di.Register(&myComponentImpl{}, di.Primary, di.Priority(2))
```

Se você deseja registrar um serviço/daemon, utilize o padrão abaixo. Esse formato só é útil para serviços que vão ser inicializado junto com o container (configuração `di.Startup(priority)`)

```go
di.Register(
    func(a DependencyA, b DependencyB) error {
        print("initialized")
    }, 
    di.Startup(100)
)
```

### Component Type = KEY & Factory Configurations

O tipo do retorno da função construtora determina o tipo do seu componente. Essa é a chave usada internamente pelo container para identificar os objetos. Você pode registrar quantos candidatos desejar para o mesmo tipo de retorno, e usar as configurações da fábrica para ajustar a prioridade, o escopo ou até mesmo qualificar algum componente, permitindo que o container possa identificar o construtor mais adequado durante o processo de injeção.

Existe um capitulo mais abaixo descrevendo as Factory Configurations existentes o como elas podem ser usadas para permitir um desenvolvimento modular da sua aplicação.

### Dependencias

Você pode usar qualquer tipo de objeto para identificar suas dependencias, porém o mais recomendado é seguir o [Princípio da Inversão de Dependencia](https://en.wikipedia.org/wiki/Dependency_inversion_principle), utilizando as `interface` e permitindo que o container possa obter a instancia compatível. Isso reduz o acoplamento entre os modulos da sua aplicação, simplificando a manutenção e testes unitários.

O container aplica a seguintes regras de priorização para determinar as fábricas candidatas para criação da dependencia.

1. Os componentes registrados com o mesmo tipo da dependencia (exact match)
2. Os componentes registrados em que a dependecia solicitada é assignable com tipo do componente registrado (see [Type.AssignableTo](https://pkg.go.dev/reflect#Type.AssignableTo), [Go Assignability](https://go.dev/ref/spec#Assignability))


Exemplo de declaraçao e uso de dependencia válida. Perceba que registramos o componente do tipo `typeof *dependencyBImpl`. Este componente é assignable com `b DependencyB` e `c *dependencyBImpl`.
```go
type DependencyA interface { DoIt() }
type DependencyB interface { DoItBetter() }

type dependencyAImpl struct { /*...implements_A_DoIt()...*/ }
type dependencyBImpl struct { /*...implements_B_DoItBetter()...*/ }

// Component Type = Key = typeof DependencyA
di.Register(func() DependencyA { return &dependencyAImpl{} })

// Component Type = Key = typeof *dependencyBImpl
di.Register(func() dependencyBImpl { return &dependencyBImpl{} })

// a Key = typeof DependencyA, exact match (di.Register)
// b Key = typeof DependencyB, assignable match ('*testServiceBImpl' is a candidate for 'testServiceB')
// c Key = typeof *dependencyBImpl, exact match (di.Register)
di.Register(func(a DependencyA, b DependencyB, c *dependencyBImpl) {    
    // b == c
})
```

Já abaixo temos um exemplo inválido de dependencia, resultado no erro `missing dependencies`.

```go
type DependencyA interface { DoIt() }

type dependencyAImpl struct { /*...implements_A_DoIt()...*/ }
func (* dependencyAImpl) DoAnotherThing() {}

type dependencyAImpl2 struct { /*...implements_A_DoIt()...*/ }

// Component Type = Key = typeof DependencyA
di.Register(func() DependencyA { return &dependencyAImpl2{} })

// d Key = typeof *dependencyAImpl
di.Register(func(d *dependencyAImpl) {    
   
})
```

Veja que apesar de termos declarado a existencia do componente do tipo `DependencyA`, e existir uma fábrica que retorna uma instancia que o tipo é assignable (`d *dependencyAImpl2`), o container não consegue saber, antes de invocar o construtor, se o tipo retornado é compatível, e neste caso não é. A nossa dependencia (`d *dependencyAImpl`) é assignable de `DependencyA`, mas também implementa outro método e poderia ser assignable de outros componentes que podem não ser satisfeito pelo contrutor existente.

Portanto, é importante que a dependencias sejam declaradas preferencialmente utilizando a interface do tipo, e não a implementação (SOLID - DIP).

## Factory Configurations

O di disponibiliza algumas configurações sobre as fábricas de componentes, permitindo que o desenvolvedor possa configurar o ciclo de vida dos componentes ou mesmo gerar identificadores para permitir a obtenção de componentes que possuam algum marcador específico.

### Initializer
Initializer register a initializer function to the component. A factory component may declare multiple initializers methods. If the factory returns nil, the initializer will be ignored


### Disposer
Disposer register a disposal function to the component. A factory component may declare multiple disposer methods. If the factory returns nil, the disposer will be ignored


### Startup
Startup indicates that this component must be initialized during container initialization (Container.Initialize method)

```go
di.Register(func()  {
	print("Second")
}, Startup(200))

di.Register(func()  {
	print("First")
}, Startup(100)) 
```

### Priority
Priority can be applied to any component to indicate in what order they should be used.

If the component is marked as Startup, the priority determines its execution order.

Priority is also used during dependency injection. The candidate with the highest priority will be injected.

A framework can implement filters and use priority to define the order of execution

### Qualify
Qualify register a qualifier for the component. Anyone can define a new qualifier.

```go
type MyQualifier string

di.Register(func() *MyService {
	return &MyService{}
}, di.Qualify[testQualifier]())
```

### Primary
Primary indicates that a component should be given preference when multiple candidates are qualified to inject a single-valued dependency. 
If exactly one 'primary' component exists among the candidates, it will be the injected value.

```go
di.Register(func(repository FooRepository) FooService {
	return &FooServiceImpl{ repository: repository }
})

di.Register(func() FooRepository {
	return &MemoryRepositoryImpl{}
})

di.Register(func() FooRepository {
	return &DatabaseRepositoryImpl{}
}, di.Primary)
```
Because DatabaseRepositoryImpl is marked with Primary, it will be injected preferentially over the MemoryRepositoryImpl variant assuming both are present as component within the same di container.

### Alternative
Alternative indicates that a component should NOT be given preference when multiple candidates are qualified to inject a single-valued dependency.

If exactly one NON-ALTERNATIVE component exists among the candidates, it will be the injected value.

```go
di.Register(func(repository FooRepository) FooService {
	return &FooServiceImpl{ repository: repository }
})

di.Register(func() FooRepository {
	return &MemoryRepositoryImpl{}
})

di.Register(func() FooRepository {
	return &DatabaseRepositoryImpl{}
}, di.Alternative)
```
Because DatabaseRepositoryImpl is marked with Alternative, it will NOT be injected over the MemoryRepositoryImpl variant assuming both are present as component within the same di container.

### Scoped
Scoped identifies the lifecycle of an instance, such as singleton, prototype, and so forth.. A scope governs how the container reuses instances of the type.

To register additional custom scopes, see Container.RegisterScope. Defaults to an empty string ("") which implies SCOPE_SINGLETON.

### Singleton (aka Scoped("singleton"))

Singleton identifies a component that only instantiates once.

```go
di.Register(func() MyService {
	return &MyServiceImpl{ Id: uuid.New() }
}, di.Singleton)

di.Register(func(s MyService) MyControllerA {
	print(s.Id) // uuid value
})

di.Register(func(s MyService) MyControllerB {
	print(s.Id) // same uuid value
})
```

### Singleton (aka Scoped("prototype"))
Prototype identifies a component that a new instance is created every time the component factory is invoked.

```go
di.Register(func() MyService {
	return &MyServiceImpl{ Id: uuid.New() }
}, di.Prototype)

di.Register(func(s MyService) MyControllerA {
	print(s.Id) // first uuid
})

di.Register(func(s MyService, ctn di.Container, ctx context.Context) MyControllerB {
	print(s.Id) // second uuid
	s2, _ := di.Get[testService](ctn, ctx)
	print(s2.Id) // third uuid
})
```

### Condition
Condition a single condition that must be matched in order for a component to be registered.

Conditions are checked immediately before the component factory is due to be registered and are free to veto registration based on any criteria that can be determined at that point.

### Stereotype
Stereotype a stereotype encapsulates any combination of ComponentOption

```go
var Controller = di.Stereotype(di.Singleton, di.Qualify[testQualifier](), di.Startup(500))

di.Register(func() MyController {
    return &MyController{}
}, Controller)
```

Example: Filter using Stereotype

```go
di.Filter(Controller).Foreach(func(f *Factory) (bool, error) { ... })
```


## Qualify

## Provider

## Unmanaged

## Scope

## Utils

## Instanciar um Componente ou Serviço na Inicializaçao

Componentes ou serviços podem ser inicialziados junto com o Container (`di.Initialize(ctx)`) por meio da configuração `di.Startup(priority)`. Isso é útil para que você possa inicializar os serviços essenciais da sua aplicação de forma sincronizada, como por exemplo:

- Executar scripts de migração de banco de dados
- Fazer o carregamento das configurações do sistema
- Registrar as suas controladores Rest ou seus endpoints
- Executar validações do ambiente

Com o uso do di é possível implementar separadamente cada um desses serviços e especificar sua ordem de execução, conforme o exemplo abaixo.


```go

// package config

di.Register(func() error {
	// we dont need to return a Component
    return doSomeEnvValidation()
}, di.Startup(0))

type JediService interface {
    FeelTheForce()
}

type PadawanController struct {
    s JediService
}

func (p * PadawanController) Initialize() {
    p.s.FeelTheForce()
}

// register as startup component, injecting dependencies
di.Register(func(s JediService) *PadawanController {
	return &PadawanController{s:s}
}, di.Startup(100))


// (...) in a package far, far away ...

type yodaServiceImp struct {}

func (s * yodaServiceImp) FeelTheForce() {
    print("Patience you must have my young Padawan")
}

di.Register(&yodaServiceImp{})

// package main

func main() {
    di.Initialize() 
}

```

