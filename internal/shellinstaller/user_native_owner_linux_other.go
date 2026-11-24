//go:build linux && !amd64

package shellinstaller

import (
	"context"
	"io"
	"time"
)

func userNativeInstall(context.Context, string, []string, UserInstallRequest, io.Reader, io.Writer, io.Writer) error {
	return privateError("unavailable", nil)
}

func userNativeOwnerRequest(context.Context, []string) (UserInstallRequest, time.Time, error) {
	return UserInstallRequest{}, time.Time{}, privateError("unavailable", nil)
}

func userNativeOwnerEntry(context.Context, string, []string, io.Reader, io.Writer, io.Writer) error {
	return privateError("unavailable", nil)
}
