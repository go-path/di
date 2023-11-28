package di

import "testing"

// Mock @TODO: Fazer mock de serviços
func Mock[T any](mock T) func() {
	if !testing.Testing() {
		panic("mocks are only allowed during testing")
	}
	return func() {
		// cleanup
	}
}
