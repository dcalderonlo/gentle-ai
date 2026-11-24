//go:build linux

package shellinstaller

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSupervisorReleaseScopedSafe(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("non-root fixture")
	}
	root := userUnixTempRoot(t)
	self := filepath.Join(root, "self")
	if err := os.WriteFile(self, []byte("inert"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx := WithSupervisorRelease(context.Background(), "1.2.3", func(context.Context, string) ([]byte, error) {
		t.Fatal("unexpected acquisition")
		return nil, nil
	})
	calls := 0
	err := WithPreparedSupervisorExecutable(ctx, self, func(path string) error {
		calls++
		if path != self {
			t.Errorf("path %q", path)
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestSupervisorReleaseScopedRootGate(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("root-only gate")
	}
	root := userUnixTempRoot(t)
	t.Setenv("TMPDIR", root)
	t.Setenv(RootRiskEnv, "")
	called := false
	err := WithPreparedSupervisorExecutable(context.Background(), filepath.Join(root, "missing"), func(string) error {
		called = true
		return nil
	})
	entries, readErr := os.ReadDir(root)
	if !AsRootRiskRequired(err) || called || readErr != nil || len(entries) != 0 {
		t.Fatalf("err=%v called=%v entries=%v read=%v", err, called, entries, readErr)
	}
}

func TestSupervisorReleaseScopedBootstrap(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("non-root Homebrew fixture")
	}
	for _, scenario := range []string{"success", "cancel-provider", "operation-error", "cancel-operation", "cancel-operation-nil", "invalid-version", "source-mutation", "identity-replacement"} {
		t.Run(scenario, func(t *testing.T) {
			root := userUnixTempRoot(t)
			t.Setenv("TMPDIR", root)
			brew := filepath.Join(root, "brew")
			if err := os.Mkdir(brew, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(brew, 0777); err != nil {
				t.Fatal(err)
			}
			self := filepath.Join(brew, "self")
			if err := os.WriteFile(self, []byte("source"), 0700); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(self)
			if err != nil {
				t.Fatal(err)
			}
			old := userHomebrewPrefixes
			userHomebrewPrefixes = []string{brew}
			t.Cleanup(func() { userHomebrewPrefixes = old })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			version := "1.2.3"
			if scenario == "invalid-version" {
				version = "latest"
			}
			acquisitions, calls := 0, 0
			ctx = WithSupervisorRelease(ctx, version, func(_ context.Context, got string) ([]byte, error) {
				acquisitions++
				if got != "1.2.3" {
					t.Errorf("version=%q", got)
				}
				if scenario == "cancel-provider" {
					cancel()
				}
				if scenario == "source-mutation" {
					if err := os.WriteFile(self, []byte("changed source"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				return []byte("verified fixture"), nil
			})
			original := &PrivateRuntimeError{Kind: "uncertain", Workspace: "original-workspace"}
			var prepared, moved string
			err = WithPreparedSupervisorExecutable(ctx, self, func(path string) error {
				calls++
				prepared = path
				data, readErr := os.ReadFile(path)
				if readErr != nil || string(data) != "verified fixture" {
					t.Errorf("payload=%q err=%v", data, readErr)
				}
				if scenario == "identity-replacement" {
					dir := filepath.Dir(path)
					moved = dir + "-moved"
					if err := os.Rename(dir, moved); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(dir, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "marker"), []byte("foreign"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if scenario == "operation-error" {
					return original
				}
				if scenario == "cancel-operation" || scenario == "cancel-operation-nil" {
					cancel()
					if scenario == "cancel-operation" {
						return ctx.Err()
					}
				}
				return nil
			})
			if scenario == "invalid-version" || scenario == "cancel-provider" || scenario == "source-mutation" {
				if calls != 0 || err == nil || (scenario == "invalid-version" && acquisitions != 0) || (scenario != "invalid-version" && acquisitions != 1) {
					t.Fatalf("acquire=%d calls=%d err=%v", acquisitions, calls, err)
				}
				if scenario == "cancel-provider" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				entries, readErr := os.ReadDir(root)
				if readErr != nil || len(entries) != 1 {
					t.Fatalf("scratch=%v err=%v", entries, readErr)
				}
				return
			}
			if acquisitions != 1 || calls != 1 {
				t.Fatalf("acquire=%d calls=%d err=%v", acquisitions, calls, err)
			}
			if scenario == "success" {
				data, readErr := os.ReadFile(self)
				after, statErr := os.Stat(self)
				if err != nil || readErr != nil || statErr != nil || string(data) != "source" || !os.SameFile(before, after) || before.Mode() != after.Mode() {
					t.Fatalf("source changed: %v %v %v", err, readErr, statErr)
				}
				if _, statErr := os.Stat(prepared); !os.IsNotExist(statErr) {
					t.Fatalf("not cleaned: %v", statErr)
				}
				return
			}
			var retained *BootstrapRetentionError
			if !errors.As(err, &retained) || retained.Path != filepath.Dir(prepared) {
				t.Fatalf("retention=%v", err)
			}
			if scenario == "identity-replacement" {
				if _, e := os.Stat(filepath.Join(retained.Path, "marker")); e != nil {
					t.Fatal(e)
				}
				if _, e := os.Stat(filepath.Join(moved, "supervisor")); e != nil {
					t.Fatal(e)
				}
				return
			}
			if _, e := os.Stat(prepared); e != nil {
				t.Fatal(e)
			}
			if scenario == "operation-error" {
				var got *PrivateRuntimeError
				if !errors.As(err, &got) || got != original || got.Workspace != "original-workspace" {
					t.Fatalf("original=%v", err)
				}
			} else if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestSupervisorReleaseMalformedInstallEntryBeforeAcquisition(t *testing.T) {
	root := userUnixTempRoot(t)
	calls := 0
	ctx := WithSupervisorRelease(context.Background(), "1.2.3", func(context.Context, string) ([]byte, error) {
		calls++
		return nil, errors.New("unexpected acquisition")
	})
	for _, command := range []string{"install", "internal-install", "internal-install-direct"} {
		err := RunUserEntry(ctx, filepath.Join(root, "missing"), []string{command}, nil, nil, nil)
		if err == nil || err.Error() != "invalid internal install arguments" {
			t.Fatalf("%s error=%v", command, err)
		}
	}
	if calls != 0 {
		t.Fatalf("malformed install acquired %d times", calls)
	}
}
