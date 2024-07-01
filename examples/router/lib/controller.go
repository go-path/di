package lib

import (
	"context"

	"github.com/go-path/di"
)

type Controller interface {
	Path() string
}

// controllersProvider list provider (list all controllers registered in container)
func controllersProvider(ctx context.Context) ([]Controller, error) {
	return di.AllOf[Controller](di.Global(), ctx)
}

func init() {
	di.Register(controllersProvider)
}
