// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// BackingServiceOptions configures a plain, uninstrumented dependency
// container (e.g. Postgres, Redis) that an AppScenario reaches by network
// alias — the same role postgres.yaml/redis.yaml play for the K8s examples
// (opted out of instrumentation there via the dash0.com/enable label).
type BackingServiceOptions struct {
	// Image is the container image to run, e.g. "postgres:16-alpine".
	Image string
	// Env sets environment variables inside the container.
	Env map[string]string
	// Network is the Docker network the app scenario also joins.
	Network string
	// Alias is the hostname the app scenario resolves this service at on
	// Network — must match what the app's own connection string/env var
	// defaults to (e.g. "postgres", "redis").
	Alias string
	// WaitingFor is the readiness strategy, e.g. wait.ForExec with the same
	// command the K8s manifest's readinessProbe uses (pg_isready, redis-cli
	// ping).
	WaitingFor wait.Strategy
}

// StartBackingService starts a dependency container on opts.Network under
// opts.Alias, for an AppScenario configured with the same Networks entry.
func StartBackingService(t testing.TB, ctx context.Context, opts BackingServiceOptions) testcontainers.Container {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:          opts.Image,
		Env:            opts.Env,
		Networks:       []string{opts.Network},
		NetworkAliases: map[string][]string{opts.Network: {opts.Alias}},
		WaitingFor:     opts.WaitingFor,
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start backing service %s", opts.Image)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	return container
}

// NewNetwork creates a Docker network for an AppScenario and its backing
// service(s) to share, torn down when the test finishes.
func NewNetwork(t testing.TB, ctx context.Context) string {
	t.Helper()
	nw, err := network.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = nw.Remove(context.Background())
	})
	return nw.Name
}
