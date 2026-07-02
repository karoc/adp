package claude

import (
	"context"
	"errors"
	"testing"

	"github.com/karoc/adp/internal/adapters/api"
)

func TestName(t *testing.T) {
	if got := New().Name(); got != Name {
		t.Fatalf("Name() = %q, want %q", got, Name)
	}
	if Name != "claude" {
		t.Fatalf("Name constant = %q, want claude", Name)
	}
}

func TestValidate(t *testing.T) {
	adapter := New()

	if err := adapter.Validate(context.Background(), claudeContext(t.TempDir())); err != nil {
		t.Fatalf("Validate() with live context error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := adapter.Validate(ctx, claudeContext(t.TempDir()))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate() with canceled context error = %v, want context.Canceled", err)
	}
}

func TestRenderRejectsCanceledContext(t *testing.T) {
	adapter := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := adapter.Render(ctx, claudeContext(t.TempDir()))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Render() error = %v, want context.Canceled", err)
	}
	if result != nil {
		t.Fatalf("Render() result = %#v, want nil on cancellation", result)
	}
}

func TestLaunchRejectsCanceledContext(t *testing.T) {
	adapter := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	spec, err := adapter.Launch(ctx, claudeContext(t.TempDir()), api.RuntimeHandle{Root: "/runtime/claude"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Launch() error = %v, want context.Canceled", err)
	}
	if spec != nil {
		t.Fatalf("Launch() spec = %#v, want nil on cancellation", spec)
	}
}
