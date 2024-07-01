package main

import (
	"github.com/go-path/di"

	_ "di/example/router/controller"
	_ "di/example/router/lib"
)

func main() {
	di.Initialize()
}
