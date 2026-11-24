//go:build linux

package cli

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v5/internal/shellinstaller"
)

func TestShellInstallReleasePreparedSelfSharedByProbeAndWorker(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("non-root inline installation fixture")
	}
	oldInspect, oldProbe, oldWorker := shellInstallInspect, shellLaunchProbe, shellDirectWorker
	oldDirect, oldDecorator := shellInlineDirect, ShellContextDecorator
	t.Cleanup(func() {
		shellInstallInspect, shellLaunchProbe, shellDirectWorker = oldInspect, oldProbe, oldWorker
		shellInlineDirect, ShellContextDecorator = oldDirect, oldDecorator
	})
	shellInlineDirect = true
	ShellContextDecorator = func(ctx context.Context) context.Context { return ctx }
	shellInstallInspect = func(shellinstaller.UserInstallRequest) (string, error) { return "token", nil }
	var probePath, workerPath string
	shellLaunchProbe = func(_ context.Context, self string) error { probePath = self; return nil }
	shellDirectWorker = func(_ context.Context, self string, _ []string, _, _ io.Writer) error { workerPath = self; return nil }
	selection := ShellInstallSelection{Confirmed: true, Request: shellinstaller.UserInstallRequest{Destination: t.TempDir(), Mode: "separate", Confirmation: "token"}}
	if err := RunShellInstallInline(context.Background(), selection, nil, io.Discard); err != nil {
		t.Fatal(err)
	}
	if probePath == "" || probePath != workerPath {
		t.Fatalf("probe=%q worker=%q", probePath, workerPath)
	}
}
