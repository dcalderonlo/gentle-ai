package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v5/internal/shellinstaller"
)

func TestShellInstallReleaseContextAtFreshAndEmbeddedEntries(t *testing.T) {
	type marker struct{}
	oldDecorator, oldInspect := ShellContextDecorator, shellInstallInspect
	t.Cleanup(func() {
		ShellContextDecorator, shellInstallInspect = oldDecorator, oldInspect
	})
	calls := 0
	ShellContextDecorator = func(ctx context.Context) context.Context {
		calls++
		return context.WithValue(ctx, marker{}, "current-build")
	}
	ctx, cancel, _, err := shellExecutionContext()
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Value(marker{}) != "current-build" {
		t.Fatal("fresh CLI context lacks application configuration")
	}
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatal("decorator lost signal-context cancellation")
	}
	if os.Geteuid() == 0 {
		t.Skip("embedded installation entry requires non-root operator consent")
	}
	shellInstallInspect = func(shellinstaller.UserInstallRequest) (string, error) { return "token", nil }
	selection := ShellInstallSelection{
		Confirmed: true,
		Request: shellinstaller.UserInstallRequest{Destination: t.TempDir(), Mode: "separate", Confirmation: "token"},
	}
	sentinel := errors.New("stop before installation")
	entered := false
	err = runShellInstallSelectionWithEntry(selection, io.Discard, func(ctx context.Context, _ string, _ []string, _ io.Reader, _, _ io.Writer) error {
		entered = true
		if ctx.Value(marker{}) != "current-build" {
			t.Error("embedded entry lost application configuration")
		}
		return sentinel
	})
	if !entered || err == nil || calls != 2 {
		t.Fatalf("embedded entry=%v calls=%d error=%v", entered, calls, err)
	}
}

func TestShellInstallReleaseInlinePreservesCallerContext(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("inline installation entry requires non-root operator consent")
	}
	type marker struct{}
	oldDecorator, oldInspect := ShellContextDecorator, shellInstallInspect
	t.Cleanup(func() {
		ShellContextDecorator, shellInstallInspect = oldDecorator, oldInspect
	})
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), marker{}, "progress-owner"))
	cancel()
	decorated := false
	ShellContextDecorator = func(got context.Context) context.Context {
		decorated = true
		if got.Err() != context.Canceled || got.Value(marker{}) != "progress-owner" {
			t.Error("inline entry lost caller cancellation or values")
		}
		return got
	}
	sentinel := errors.New("stop at inspection")
	shellInstallInspect = func(shellinstaller.UserInstallRequest) (string, error) { return "", sentinel }
	err := RunShellInstallInline(ctx, ShellInstallSelection{Confirmed: true}, nil, io.Discard)
	if !decorated || !errors.Is(err, sentinel) {
		t.Fatalf("decorated=%v error=%v", decorated, err)
	}
}
