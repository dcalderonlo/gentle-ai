package shellinstaller

import (
	"context"
	"testing"
)

func TestSupervisorReleaseContextPreservesCancellationAndVersion(t *testing.T) {
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	ctx := WithSupervisorRelease(base, "1.2.3", func(ctx context.Context, version string) ([]byte, error) {
		called = true
		if version != "1.2.3" {
			t.Fatalf("version = %q", version)
		}
		return []byte("verified"), ctx.Err()
	})
	cfg, ok := supervisorReleaseFromContext(ctx)
	if !ok || cfg.version != "1.2.3" {
		t.Fatal("release configuration missing")
	}
	if _, err := cfg.acquire(ctx, cfg.version); err != nil || !called {
		t.Fatalf("acquisition: %v", err)
	}
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatal("cancellation not preserved")
	}
}
