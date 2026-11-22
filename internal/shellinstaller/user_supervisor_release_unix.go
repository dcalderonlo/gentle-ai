//go:build linux || darwin

package shellinstaller

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// StageUserSupervisor provisions a supervisor into an absent destination in an
// owned private directory. Existing destinations are never replaced. Source
// admission is unchanged; only a classified Homebrew ancestor refusal may use
// the application's authenticated exact-release acquisition configuration.
func StageUserSupervisor(ctx context.Context, source, dest string) (digest string, err error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	parent := filepath.Dir(dest)
	if _, err := privateDirectory(parent); err != nil {
		return "", err
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		return "", fmt.Errorf("supervisor destination must be absent: %s", dest)
	}
	stage, err := os.MkdirTemp(parent, ".supervisor-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	candidate := filepath.Join(stage, "supervisor")
	digest, err = userPrepareSupervisor(ctx, source, candidate)
	if err != nil {
		return "", err
	}
	readback, err := userSupervisorSHA(ctx, candidate)
	if err != nil {
		return "", err
	}
	if readback != digest {
		return "", errors.New("staged supervisor checksum mismatch")
	}
	identity, err := os.Lstat(candidate)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Link is an atomic no-clobber publication on the same private filesystem.
	// Unlike Rename, it cannot overwrite a destination appearing after Lstat.
	if err := os.Link(candidate, dest); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			current, statErr := os.Lstat(dest)
			if statErr == nil && os.SameFile(identity, current) {
				err = errors.Join(err, os.Remove(dest), userDirectorySync(parent))
			}
			digest = ""
		}
	}()
	if err = userDirectorySync(parent); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return digest, nil
}

func userPrepareSupervisor(ctx context.Context, source, dest string) (string, error) {
	sha, err := userCopySupervisor(ctx, source, dest)
	if err == nil {
		return sha, nil
	}
	var refusal *supervisorAncestorRefusal
	if !errors.As(err, &refusal) || userHomebrewHint(refusal.path) == "" {
		return "", err
	}
	cfg, ok := supervisorReleaseFromContext(ctx)
	if !ok || cfg.acquire == nil {
		return "", fmt.Errorf("authenticated supervisor acquisition unavailable; run a trusted release from a private directory: %w", err)
	}
	stable, _ := regexp.MatchString(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`, cfg.version)
	if !stable {
		return "", fmt.Errorf("supervisor version %q is not exact stable semver; run an official stable release from a private directory", cfg.version)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	payload, acquireErr := cfg.acquire(ctx, cfg.version)
	if acquireErr != nil {
		return "", fmt.Errorf("acquire authenticated supervisor for %s; run a trusted release from a private directory: %w", cfg.version, acquireErr)
	}
	if len(payload) == 0 {
		return "", errors.New("authenticated supervisor acquisition returned empty payload")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(payload))
	if err := userToolWrite(dest, payload, 0700); err != nil {
		return "", err
	}
	actual, err := userSupervisorSHA(ctx, dest)
	if err != nil {
		return "", fmt.Errorf("supervisor readback: %w", err)
	}
	if actual != expected {
		return "", errors.New("authenticated supervisor readback checksum mismatch")
	}
	return actual, nil
}
