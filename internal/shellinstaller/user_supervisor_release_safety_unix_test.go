//go:build linux || darwin

package shellinstaller

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStageUserSupervisorPreservesExistingDestination(t *testing.T) {
	root := userUnixTempRoot(t)
	source, dest := filepath.Join(root, "source"), filepath.Join(root, "supervisor")
	for _, name := range []string{source, dest} {
		if err := os.WriteFile(name, []byte("prior"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	ctx := WithSupervisorRelease(context.Background(), "1.2.3", func(context.Context, string) ([]byte, error) {
		t.Error("existing destination must refuse before acquisition")
		return nil, errors.New("unexpected acquisition")
	})
	got, err := StageUserSupervisor(ctx, source, dest)
	if err == nil || got != "" {
		t.Fatalf("existing destination accepted: %q %v", got, err)
	}
	for _, name := range []string{source, dest} {
		data, err := os.ReadFile(name)
		if err != nil || string(data) != "prior" {
			t.Fatalf("prior bytes changed: %q %v", data, err)
		}
	}
}

func TestStageUserSupervisorRejectsMalformedVersionAndCanceledAcquisition(t *testing.T) {
	root := userUnixTempRoot(t)
	brew := filepath.Join(root, "brew")
	if err := os.Mkdir(brew, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(brew, 0777); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(brew, "gentle-ai")
	if err := os.WriteFile(source, []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	old := userHomebrewPrefixes
	userHomebrewPrefixes = []string{brew}
	t.Cleanup(func() { userHomebrewPrefixes = old })
	for _, version := range []string{"dev", "latest", "v1.2.3", "1.2.3-rc.1", "1.2.3"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			ctx = WithSupervisorRelease(ctx, version, func(context.Context, string) ([]byte, error) {
				calls++
				cancel()
				return []byte("must not publish"), nil
			})
			dest := filepath.Join(root, "supervisor")
			got, err := StageUserSupervisor(ctx, source, dest)
			if err == nil || got != "" {
				t.Fatalf("invalid/canceled acquisition accepted: %q %v", got, err)
			}
			if version != "1.2.3" && calls != 0 {
				t.Fatal("malformed version reached acquisition")
			}
			if version == "1.2.3" && calls != 1 {
				t.Fatal("valid version did not reach cancellation fixture")
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatal("failed acquisition published destination")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != "brew" {
				t.Fatalf("owned scratch leaked: %v %v", entries, err)
			}
		})
	}
}

func TestStageUserSupervisorRejectsUnsafeDestinationParent(t *testing.T) {
	root := userUnixTempRoot(t)
	source := filepath.Join(root, "source")
	if err := os.WriteFile(source, []byte("original"), 0700); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "unsafe")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := StageUserSupervisor(context.Background(), source, filepath.Join(parent, "supervisor")); err == nil {
		t.Fatal("unsafe destination parent accepted")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := StageUserSupervisor(context.Background(), source, filepath.Join(link, "supervisor")); err == nil {
		t.Fatal("symlink destination parent accepted")
	}
}
