package platform

import (
	"fmt"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// Factory creates an Adapter for a given platform model.
type Factory func(platform *models.Platform) (Adapter, error)

// Registry maps platform types to adapter factories.
type Registry struct {
	factories map[v1.PlatformType]Factory
}

// NewRegistry creates an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[v1.PlatformType]Factory),
	}
}

// Register adds a factory for a platform type.
func (r *Registry) Register(t v1.PlatformType, f Factory) {
	r.factories[t] = f
}

// Create builds an adapter for the given platform.
func (r *Registry) Create(platform *models.Platform) (Adapter, error) {
	f, ok := r.factories[v1.PlatformType(platform.Type)]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for platform type %d", platform.Type)
	}
	return f(platform)
}
