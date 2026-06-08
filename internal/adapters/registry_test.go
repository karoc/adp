package adapters

import (
	"reflect"
	"testing"
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
