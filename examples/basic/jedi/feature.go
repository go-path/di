package jedi

import "github.com/go-path/di"

type yodaServiceImp struct{}

func (s *yodaServiceImp) FeelTheForce() {
	println("[Yoda] Patience You Must Have My Young Padawan")
}

func init() {
	di.Register(&yodaServiceImp{})
}
