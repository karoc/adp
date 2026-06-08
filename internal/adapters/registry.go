package adapters

import (
	"fmt"
	"sort"

	"github.com/karoc/adp/internal/adapters/claude"
	"github.com/karoc/adp/internal/adapters/codex"
)

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: map[string]Adapter{}}
}

func NewDefaultRegistry() (*Registry, error) {
	registry := NewRegistry()
	if err := RegisterDefaults(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func RegisterDefaults(registry *Registry) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}
	for _, adapter := range []Adapter{
		codex.New(),
		claude.New(),
	} {
		if err := registry.Register(adapter); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	name := adapter.Name()
	if name == "" {
		return fmt.Errorf("adapter name is required")
	}
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter %q already registered", name)
	}
	r.adapters[name] = adapter
	return nil
}

func (r *Registry) Get(name string) (Adapter, bool) {
	adapter, ok := r.adapters[name]
	return adapter, ok
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
