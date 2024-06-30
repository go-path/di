<a id="go-path-di"></a>
# go-path/di

This is The Way to do Dependency injection in Go.

> **Don't know what DI is?** [Wikipedia](https://en.wikipedia.org/wiki/Dependency_injection)


## Show me the code!

Source: https://github.com/go-path/di/tree/main/examples/basic


```go
package padawan

import "github.com/go-path/di"

type JediService interface {
	FeelTheForce()
}

type PadawanController struct {
	Master JediService `inject:""`
}

func (p *PadawanController) Initialize() {
	println("[Padawan] Master, I want to learn the ways of the force...")
	p.Master.FeelTheForce()
}

func init() {
	// register as startup component, injecting dependencies
	di.Register(di.Injected[*PadawanController](), di.Startup(100))
}
```

```go
package jedi

import "github.com/go-path/di"

type yodaServiceImp struct{}

func (s *yodaServiceImp) FeelTheForce() {
	println("[Yoda] Patience You Must Have My Young Padawan")
}

func init() {
	di.Register(&yodaServiceImp{})
}
```

```go
package main

import (
	"github.com/go-path/di"

	_ "di/example/basic/jedi"
	_ "di/example/basic/padawan"
)

func main() {
	di.Initialize()
}
```

**output**
```shell
INFO [di] '*jedi.yodaServiceImp' is a candidate for 'padawan.JediService'
[Padawan] Master, I want to learn the ways of the force...
[Yoda] Patience You Must Have My Young Padawan
```

## Features

- **Typesafe**: Using generics.
- **Extensible**: And configurable.
- **Ergonomic**: Easy to understand. Focused on providing a simple yet efficient API.
- **Predictable**: The programmer has control over dependencies and can define qualifiers.
- **Framework Agnostic**: It can be used with any library.

## Goal

To be a lightweight, simple, ergonomic and high-performance DI container that can be easily extended. A foundation for the development of more complex frameworks and architectural structures.

## Usage

Read our [Full Documentation][docs] to learn how to use **go-path/di**.

## Warning
> ARE YOU ALLERGIC?
> 
> Some concepts applied in this library may contain gluten and/or be inspired by the implementations of [CDI Java](https://www.cdi-spec.org/) and [Spring IoC](https://docs.spring.io/spring-framework/reference/core/beans.html).
> 
> And obviously, this implementation uses [reflection](https://pkg.go.dev/reflect), which for some uninformed individuals can be detrimental. It is important to mention that we thoroughly sanitize our hands before handling any `Type` or `Value`.
>
> We will not be held responsible if you become a productive developer after coming into contact with any part of this **go-path/di**.
>
> "Your path you must decide.” — Yoda

## Get involved
All kinds of contributions are welcome!

🐛 **Found a bug?**  
Let me know by [creating an issue][new-issue].

❓ **Have a question?**  
[Discussions][discussions] is a good place to start.

⚙️ **Interested in fixing a [bug][bugs] or adding a [feature][features]?**  
Check out the [contributing guidelines](CONTRIBUTING.md).

📖 **Can we improve [our documentation][docs]?**  
Pull requests even for small changes can be helpful. Each page in the docs can be edited by clicking the 
"Edit on GitHub" link at the bottom right.

[docs]: https://go-path.github.io/di
[bugs]: https://github.com/go-path/di/issues?q=is%3Aissue+is%3Aopen+label%3Abug
[features]: https://github.com/go-path/di/issues?q=is%3Aissue+is%3Aopen+label%3Afeature
[new-issue]: https://github.com/go-path/di/issues/new/choose
[discussions]: https://github.com/go-path/di/discussions

## License

This code is distributed under the terms and conditions of the [MIT license](LICENSE).




