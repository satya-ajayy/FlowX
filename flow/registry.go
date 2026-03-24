package flow

import "fmt"

// registry maps config-friendly flow names to their code-defined definitions.
// Add new flows here when they are created.
var registry = map[string]Flow{
	"dummy_flow": Dummy,
}

// Exists checks if a flow with the given name is registered.
func Exists(name string) bool {
	_, ok := registry[name]
	return ok
}

// Get returns the flow matching the given name.
// Returns an error if no flow is registered with that name.
func Get(name string) (Flow, error) {
	f, ok := registry[name]
	if !ok {
		return Flow{}, fmt.Errorf("%s not found in registry", name)
	}
	return f, nil
}
