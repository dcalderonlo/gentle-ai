//go:build linux

package shellinstaller

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Cancellation inside acquisition makes the required pre-admission behavior
// observable without executing the fixture bytes or asking systemd to run them.
func TestSupervisorReleaseProbePreparesBeforeAdmission(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("non-root probe fixture; accepted-root service admission requires separate qualification")
	}
	root := userUnixTempRoot(t)
	t.Setenv("TMPDIR", root)
	brew := filepath.Join(root, "brew")
	if err := os.Mkdir(brew, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(brew, 0777); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(brew, "gentle-ai")
	original := []byte("inert source fixture; never execute")
	if err := os.WriteFile(source, original, 0700); err != nil {
		t.Fatal(err)
	}
	oldPrefixes := userHomebrewPrefixes
	userHomebrewPrefixes = []string{brew}
	t.Cleanup(func() { userHomebrewPrefixes = oldPrefixes })

	// Establish that the real, unchanged guard reaches the intended ancestor,
	// not a file-mode or owner error that would make the regression ambiguous.
	_, guardErr := userSupervisorSHA(context.Background(), source)
	var refusal *supervisorAncestorRefusal
	if !errors.As(guardErr, &refusal) || refusal.path != brew {
		t.Fatalf("fixture did not reach the Homebrew ancestor guard: %v", guardErr)
	}
	beforeSource, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	ctx = WithSupervisorRelease(ctx, "1.2.3", func(got context.Context, version string) ([]byte, error) {
		calls++
		if version != "1.2.3" {
			t.Errorf("acquisition version = %q; want exact 1.2.3", version)
		}
		if got.Err() != nil {
			t.Errorf("acquisition began with canceled context: %v", got.Err())
		}
		cancel()
		return []byte("inert authenticated fixture; never execute"), nil
	})
	err = RunUserLaunchProbe(ctx, source)
	if calls != 1 {
		t.Errorf("acquisition calls = %d; want 1 before service admission", calls)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("probe error = %v; want context.Canceled before execution", err)
	}
	data, readErr := os.ReadFile(source)
	afterSource, statErr := os.Stat(source)
	if readErr != nil || statErr != nil || string(data) != string(original) {
		t.Fatalf("source changed: bytes=%q read=%v stat=%v", data, readErr, statErr)
	}
	if !os.SameFile(beforeSource, afterSource) || beforeSource.Mode() != afterSource.Mode() {
		t.Error("probe replaced source identity or permissions")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "brew" {
		t.Fatalf("probe left scratch, workspace or publication in owned root: %v %v", entries, err)
	}
	children, err := os.ReadDir(brew)
	if err != nil || len(children) != 1 || children[0].Name() != "gentle-ai" {
		t.Fatalf("probe changed source directory: %v %v", children, err)
	}
}
