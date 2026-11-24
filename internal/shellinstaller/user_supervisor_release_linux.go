//go:build linux

package shellinstaller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// BootstrapRetentionError identifies a bootstrap that must be inspected or retained.
type BootstrapRetentionError struct {
	Path  string
	Cause error
}

func (e *BootstrapRetentionError) Error() string {
	return fmt.Sprintf("supervisor bootstrap retained or cleanup uncertain at %q: %v", e.Path, e.Cause)
}

func (e *BootstrapRetentionError) Unwrap() error { return e.Cause }

// WithPreparedSupervisorExecutable keeps a verified bootstrap alive for the operation.
func WithPreparedSupervisorExecutable(ctx context.Context, self string, operation func(string) error) error {
	if ctx == nil || operation == nil {
		return errors.New("supervisor context and operation required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := RootRiskGate(); err != nil {
		return err
	}
	_, err := userSupervisorSHA(ctx, self)
	if err == nil {
		return operation(self)
	}
	var refusal *supervisorAncestorRefusal
	if !errors.As(err, &refusal) || userHomebrewHint(refusal.path) == "" {
		return err
	}
	// Root never acquires or executes from a user-writable ancestry.
	if os.Getuid() == 0 {
		for path := self; ; path = filepath.Dir(path) {
			info, statErr := os.Lstat(path)
			if statErr != nil {
				return statErr
			}
			if info.Sys().(*syscall.Stat_t).Uid != 0 || info.Mode().Perm()&0022 != 0 {
				return err
			}
			if path == "/" {
				break
			}
		}
	}
	sourceIdentity, err := privateForeignPhysical(self)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", ".gentle-supervisor-")
	if err != nil {
		return err
	}
	identity, err := privateDirectory(dir)
	if err != nil {
		return &BootstrapRetentionError{Path: dir, Cause: err}
	}
	cleanup := func(cause error) error {
		if cleanupErr := privateCleanup(dir, identity); cleanupErr != nil {
			return &BootstrapRetentionError{Path: dir, Cause: errors.Join(cause, cleanupErr)}
		}
		return cause
	}
	prepared := filepath.Join(dir, "supervisor")
	if _, err := StageUserSupervisor(ctx, self, prepared); err != nil {
		return cleanup(err)
	}
	_, sourceErr := userSupervisorSHA(ctx, self)
	var again *supervisorAncestorRefusal
	if !errors.As(sourceErr, &again) || userHomebrewHint(again.path) == "" || again.path != refusal.path {
		return cleanup(fmt.Errorf("supervisor source admission changed: %w", sourceErr))
	}
	fresh, err := privateForeignPhysical(self)
	if err != nil {
		return cleanup(err)
	}
	if privateStamp(sourceIdentity) != privateStamp(fresh) {
		return cleanup(errors.New("supervisor source changed during preparation"))
	}
	if err := ctx.Err(); err != nil {
		return cleanup(err)
	}
	if err := operation(prepared); err != nil {
		return &BootstrapRetentionError{Path: dir, Cause: err}
	}
	if err := ctx.Err(); err != nil {
		return &BootstrapRetentionError{Path: dir, Cause: err}
	}
	return cleanup(nil)
}
