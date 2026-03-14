package reporter

import (
	"fmt"

	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// Registry maps reporter types to handlers.
type Registry struct {
	handlers map[v1.ReporterType]Handler
}

// NewRegistry creates an empty reporter registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[v1.ReporterType]Handler),
	}
}

// Register adds a handler for a reporter type.
func (r *Registry) Register(h Handler) {
	r.handlers[h.Type()] = h
}

// Get returns the handler for a reporter type.
func (r *Registry) Get(t v1.ReporterType) (Handler, error) {
	h, ok := r.handlers[t]
	if !ok {
		return nil, fmt.Errorf("no handler registered for reporter type %d", t)
	}
	return h, nil
}
