# Factory Config

**go-path/di** provides several configurations for component factories, allowing the developer to configure the life cycle of components or even generate identifiers to allow obtaining components that have a specific marker.

## Startup

Startup indicates that this component must be initialized during container initialization (`di.Initialize` method)

```go
di.Register(func()  {
	print("Second")
}, Startup(200))

di.Register(func()  {
	print("First")
}, Startup(100)) 
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


## Initializer
Initializer register a initializer function to the component. A factory component may declare multiple initializers methods. If the factory returns nil, the initializer will be ignored

### Disposer
Disposer register a disposal function to the component. A factory component may declare multiple disposer methods. If the factory returns nil, the disposer will be ignored

### Order
Order can be applied to any component to indicate in what order they should be used.

Higher values are interpreted as lower priority. As a consequence, the object with the lowest value has the highest priority.

Same order values will result in arbitrary sort positions for the affected objects.

If the component is marked as Startup, the order determines its execution order.

Order is also used during dependency injection. The candidate with the lower order will be injected.

A framework can implement filters and use order to define the order of execution

### Qualify
Qualify register a qualifier for the component. Anyone can define a new qualifier.

```go
type MyQualifier string

di.Register(func() *MyService {
	return &MyService{}
}, di.Qualify[testQualifier]())
```


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

__UNDER_CONSTRUCTION__

## Provider

__UNDER_CONSTRUCTION__

## Unmanaged

__UNDER_CONSTRUCTION__

## Scope

__UNDER_CONSTRUCTION__

## Utils


