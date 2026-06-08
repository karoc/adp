package adapters

import (
	"context"
	"reflect"
	"testing"

	"github.com/karoc/adp/internal/adapters/api"
)

func TestNewDefaultRegistryNames(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() error = %v", err)
	}

	got := registry.Names()
	want := []string{"claude", "codex"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registry names = %v, want %v", got, want)
	}

	for _, name := range want {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("registry.Get(%q) was not registered", name)
		}
	}
}

func TestRegisterDefaultsRejectsNilRegistry(t *testing.T) {
	if err := RegisterDefaults(nil); err == nil {
		t.Fatal("RegisterDefaults(nil) error = nil, want error")
	}
}

func TestRegistryAcceptsFutureAdapterNames(t *testing.T) {
	registry := NewRegistry()
	adapter := registryTestAdapter{name: "future-agent"}
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("Register(future-agent) error = %v", err)
	}

	got, ok := registry.Get("future-agent")
	if !ok {
		t.Fatal("Get(future-agent) ok = false")
	}
	if got != adapter {
		t.Fatalf("Get(future-agent) = %T, want registered adapter", got)
	}
	if !reflect.DeepEqual(registry.Names(), []string{"future-agent"}) {
		t.Fatalf("Names() = %v", registry.Names())
	}
}

func TestRegistryRejectsInvalidAdapterNames(t *testing.T) {
	for _, name := range []string{" future", "../future", "future/agent", "-future", ""} {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			if err := registry.Register(registryTestAdapter{name: name}); err == nil {
				t.Fatalf("Register(%q) error = nil, want error", name)
			}
		})
	}
}

func TestRegistryRejectsDuplicateAdapters(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(registryTestAdapter{name: "future-agent"}); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if err := registry.Register(registryTestAdapter{name: "future-agent"}); err == nil {
		t.Fatal("Register(duplicate) error = nil, want error")
	}
}

type registryTestAdapter struct {
	name string
}

func (a registryTestAdapter) Name() string {
	return a.name
}

func (a registryTestAdapter) Validate(context.Context, api.Context) error {
	return nil
}

func (a registryTestAdapter) Render(context.Context, api.Context) (*api.RenderResult, error) {
	return &api.RenderResult{}, nil
}

func (a registryTestAdapter) Launch(context.Context, api.Context, api.RuntimeHandle, []string) (*api.LaunchSpec, error) {
	return &api.LaunchSpec{}, nil
}
