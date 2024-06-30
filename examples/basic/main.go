package main

import (
	"github.com/go-path/di"

	_ "di/example/basic/jedi"
	_ "di/example/basic/padawan"
)

func main() {
	di.Initialize()
}
