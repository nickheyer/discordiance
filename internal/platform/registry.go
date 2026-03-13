package platform

import (
	"fmt"
	"sync"
)

// Factory creates a new Platform instance.
type Factory func() Platform

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a platform factory to the registry. Typically called from init().
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("platform already registered: %s", name))
	}
	registry[name] = factory
}

// Create instantiates a platform by its registered name.
func Create(name string) (Platform, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unknown platform type: %s", name)
	}
	return factory(), nil
}

// Available returns all registered platform type names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
