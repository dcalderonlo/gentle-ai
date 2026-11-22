//go:build linux || darwin

package shellinstaller

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSupervisorReleaseStaging(t *testing.T) {
	root := userUnixTempRoot(t)
	unsafe := filepath.Join(root, "brew")
	if err := os.Mkdir(unsafe, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafe, 0777); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(unsafe, "gentle-ai")
	if err := os.WriteFile(source, []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	old := userHomebrewPrefixes
	userHomebrewPrefixes = []string{unsafe}
	t.Cleanup(func() { userHomebrewPrefixes = old })
	for _, name := range []string{"missing", "error", "cancel", "empty", "success"} {
		t.Run(name, func(t *testing.T) {
			dest := filepath.Join(root, name)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if name == "cancel" {
				cancel()
			}
			calls := 0
			if name != "missing" {
				ctx = WithSupervisorRelease(ctx, "1.2.3", func(_ context.Context, version string) ([]byte, error) {
					calls++
					if version != "1.2.3" {
						t.Fatal(version)
					}
					if name == "error" {
						return nil, errors.New("unavailable")
					}
					if name == "empty" {
						return nil, nil
					}
					return []byte("authenticated"), nil
				})
			}
			got, err := StageUserSupervisor(ctx, source, dest)
			if name != "success" {
				if err == nil {
					t.Fatal("expected failure")
				}
				if _, err := os.Stat(dest); !os.IsNotExist(err) {
					t.Fatal("destination exists")
				}
				if name == "cancel" && calls != 0 {
					t.Fatal("canceled staging acquired release")
				}
			} else {
				if err != nil || calls != 1 || got != fmt.Sprintf("%x", sha256.Sum256([]byte("authenticated"))) {
					t.Fatalf("sha=%s calls=%d err=%v", got, calls, err)
				}
				data, err := os.ReadFile(dest)
				if err != nil || string(data) != "authenticated" {
					t.Fatalf("staged %q: %v", data, err)
				}
			}
			data, err := os.ReadFile(source)
			if err != nil || string(data) != "original" {
				t.Fatalf("source %q: %v", data, err)
			}
		})
	}
}

func TestSupervisorReleasePreservesPrivateAndUnrelatedSourcePolicy(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		t.Run(fmt.Sprintf("unsafe=%v", unsafe), func(t *testing.T) {
			root := userUnixTempRoot(t)
			sourceDir := filepath.Join(root, "source")
			if err := os.Mkdir(sourceDir, 0700); err != nil {
				t.Fatal(err)
			}
			if unsafe {
				if err := os.Chmod(sourceDir, 0777); err != nil {
					t.Fatal(err)
				}
			}
			source := filepath.Join(sourceDir, "gentle-ai")
			if err := os.WriteFile(source, []byte("original"), 0700); err != nil {
				t.Fatal(err)
			}
			ctx := WithSupervisorRelease(context.Background(), "1.2.3", func(context.Context, string) ([]byte, error) {
				t.Error("acquisition must not run for private or unrelated source")
				return nil, errors.New("unexpected acquisition")
			})
			dest := filepath.Join(root, "supervisor")
			got, err := StageUserSupervisor(ctx, source, dest)
			if unsafe {
				if err == nil {
					t.Fatal("unrelated unsafe source accepted")
				}
				if _, err := os.Stat(dest); !os.IsNotExist(err) {
					t.Fatal("unrelated source produced staged supervisor")
				}
			} else if err != nil || got != fmt.Sprintf("%x", sha256.Sum256([]byte("original"))) {
				t.Fatalf("private source changed: sha=%s error=%v", got, err)
			}
		})
	}
}
