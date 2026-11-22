package shellinstaller

import "context"

// SupervisorReleaseAcquire obtains authenticated bytes for an exact build version.
type SupervisorReleaseAcquire func(context.Context, string) ([]byte, error)

type supervisorReleaseConfig struct {
	version string
	acquire SupervisorReleaseAcquire
}

type supervisorReleaseKey struct{}

// WithSupervisorRelease carries trusted application configuration into an installer run.
func WithSupervisorRelease(ctx context.Context, version string, acquire SupervisorReleaseAcquire) context.Context {
	return context.WithValue(ctx, supervisorReleaseKey{}, supervisorReleaseConfig{version: version, acquire: acquire})
}

func supervisorReleaseFromContext(ctx context.Context) (supervisorReleaseConfig, bool) {
	cfg, ok := ctx.Value(supervisorReleaseKey{}).(supervisorReleaseConfig)
	return cfg, ok
}
