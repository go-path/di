<a id="go-path-di"></a>
# go-path/di

This is The Way to do Dependency injection in Go.

> **Don't know what DI is?** [Wikipedia](https://en.wikipedia.org/wiki/Dependency_injection)

# Installation

`go get github.com/go-path/di` 

# Show me the code!

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


# Features

- **Typesafe**: Using generics.
- **Extensible**: And configurable.
- **Ergonomic**: Easy to understand. Focused on providing a simple yet efficient API.
- **Predictable**: The programmer has control over dependencies and can define qualifiers.
- **Framework Agnostic**: It can be used with any library.

# Goal

To be a lightweight, simple, ergonomic and high-performance DI container that can be easily extended. A foundation for the development of more complex frameworks and architectural structures.

# Next steps

You can browse or search for specific subjects in the side menu. Here are some relevant links:

<br>
<br>
<div class="home-row clearfix" style="text-align:center">
    <div class="home-col">
      <div class="panel home-panel">
         <div class="panel-body">
            <p>
                <a href="#/concepts?id=general-concepts">
                    <img src="/assets/icon-parts.png" data-origin="assets/icon-parts.png" alt="General Concepts" data-no-zoom>
                </a>
            </p>
         </div>
         <div class="panel-heading">
            <p><a href="#/concepts?id=general-concepts">General Concepts</a></p>
         </div>
      </div>
   </div>
   <div class="home-col">
      <div class="panel home-panel">
         <div class="panel-body">
            <p> 
                <a href="#/component?id=component">
                    <img src="/assets/icon-component.png" data-origin="assets/icon-component.png" alt="Components" data-no-zoom>
                </a>
            </p>
         </div>
         <div class="panel-heading">
            <p><a href="#/component?id=component">Components</a></p>
         </div>
      </div>
   </div>
   <div class="home-col">
      <div class="panel home-panel">
         <div class="panel-body">
            <p>
                <a href="#/factory?id=factory-config">
                    <img src="/assets/icon-config.png" alt="Factory Config" data-no-zoom data-origin="assets/icon-config.png">
                </a>
            </p>
         </div>
         <div class="panel-heading">
            <p><a href="#/factory?id=factory-config">Factory Config</a></p>
         </div>
      </div>
   </div>
   <div class="home-col">
      <div class="panel home-panel">
         <div class="panel-body">
            <p>
                <a href="#/examples">
                    <img src="/assets/icon-tutorial.png" data-origin="assets/icon-tutorial.png" alt="Examples" data-no-zoom="">
                </a>
            </p>
         </div>
         <div class="panel-heading">
            <p><a href="#/examples">Examples</a></p>
         </div>
      </div>
   </div>
</div>
