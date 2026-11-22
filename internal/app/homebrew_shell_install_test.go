package app

import (
	"context"
	"io"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v5/internal/cli"
	"github.com/gentleman-programming/gentle-ai/v5/internal/update/upgrade"
)

func TestShellInstallInitializesBuildBeforeEarlyDispatch(t *testing.T) {
	oldVersion, oldCLI, oldUpgrade := Version, cli.AppVersion, upgrade.AppVersion
	oldDecorator := cli.ShellContextDecorator
	t.Cleanup(func() {
		Version, cli.AppVersion, upgrade.AppVersion = oldVersion, oldCLI, oldUpgrade
		cli.ShellContextDecorator = oldDecorator
	})
	// Each invocation models initialization in a fresh executable, including a
	// worker reentry. Invalid internal arguments refuse before installation.
	for _, command := range []string{"--help", "internal-install-direct"} {
		Version = "5.0.0"
		cli.AppVersion, upgrade.AppVersion = "stale", "stale"
		cli.ShellContextDecorator = nil
		err := RunArgs([]string{"shell", command}, io.Discard)
		if command == "--help" && err != nil {
			t.Fatal(err)
		}
		if command == "internal-install-direct" && err == nil {
			t.Fatal("worker accepted missing installation arguments")
		}
		if cli.AppVersion != Version || upgrade.AppVersion != Version || cli.ShellContextDecorator == nil {
			t.Fatal("early shell dispatch did not initialize the current build")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if cli.ShellContextDecorator(ctx).Err() != context.Canceled {
			t.Fatal("application release context lost cancellation")
		}
	}
}
