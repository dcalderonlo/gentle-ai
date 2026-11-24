//go:build linux && amd64

package shellinstaller

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSupervisorReleaseCanceledNativeOwnerEntryBeforeAcquisition(t *testing.T) {
	root := userUnixTempRoot(t)
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	ctx = WithSupervisorRelease(ctx, "1.2.3", func(context.Context, string) ([]byte, error) {
		calls++
		return nil, errors.New("unexpected acquisition")
	})
	cancel()
	var stdout, stderr bytes.Buffer
	err := RunUserEntry(ctx, filepath.Join(root, "missing"), []string{"internal-native-own"}, nil, &stdout, &stderr)
	if !errors.Is(err, context.Canceled) || calls != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("error=%v acquisitions=%d stdout=%q stderr=%q", err, calls, stdout.String(), stderr.String())
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("canceled entry changed scratch: entries=%v error=%v", entries, readErr)
	}
}

func TestSupervisorReleaseMalformedNativeOwnerEntryBeforeAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing-arguments", nil, "invalid native owner arguments"},
		{"invalid-deadline", []string{"invalid", "", "", "", "", ""}, "invalid native operation deadline"},
		{"expired-deadline", []string{strconv.FormatInt(time.Now().Add(-time.Minute).UnixNano(), 10), "", "", "", "", ""}, "invalid native operation deadline"},
		{"unbounded-deadline", []string{strconv.FormatInt(time.Now().Add(time.Hour).UnixNano(), 10), "", "", "", "", ""}, "invalid native operation deadline"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := userUnixTempRoot(t)
			calls := 0
			ctx := WithSupervisorRelease(context.Background(), "1.2.3", func(context.Context, string) ([]byte, error) {
				calls++
				return nil, errors.New("unexpected acquisition")
			})
			var stdout, stderr bytes.Buffer
			args := append([]string{"internal-native-own"}, tc.args...)
			err := RunUserEntry(ctx, filepath.Join(root, "missing"), args, nil, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if calls != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("acquisitions=%d stdout=%q stderr=%q", calls, stdout.String(), stderr.String())
			}
			entries, readErr := os.ReadDir(root)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("rejected entry changed scratch: entries=%v error=%v", entries, readErr)
			}
		})
	}
}
