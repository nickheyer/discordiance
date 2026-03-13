package reporter

import (
	"fmt"
	"sync"
)

// Factory creates a new Reporter instance.
type Factory func() Reporter

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a reporter factory to the registry. Typically called from init().
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("reporter already registered: %s", name))
	}
	registry[name] = factory
}

// Create instantiates a reporter by its registered name.
func Create(name string) (Reporter, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unknown reporter type: %s", name)
	}
	return factory(), nil
}

// Available returns all registered reporter type names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
