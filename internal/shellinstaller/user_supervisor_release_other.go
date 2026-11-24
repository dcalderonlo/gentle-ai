//go:build !linux

package shellinstaller

import (
	"context"
	"errors"
)

// WithPreparedSupervisorExecutable preserves existing non-Linux execution.
func WithPreparedSupervisorExecutable(ctx context.Context, self string, operation func(string) error) error {
	if ctx == nil || operation == nil {
		return errors.New("supervisor context and operation required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return operation(self)
}
