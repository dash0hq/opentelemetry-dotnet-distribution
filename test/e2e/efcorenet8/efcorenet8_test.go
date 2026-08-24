// SPDX-License-Identifier: Apache-2.0

// Package efcorenet8 is the net8.0 counterpart of the efcore scenario. It
// exists to check whether the missing-database-span issue found there is
// specific to the Dash0 distro / net6.0, or reproduces with any
// OTEL_DOTNET_AUTO_HOME-compatible tracer-home (including the actual
// upstream open-telemetry/opentelemetry-dotnet-instrumentation release) on
// a modern .NET version with a current EF Core/Npgsql pairing.
//
// To check against upstream specifically, point DASH0_E2E_TRACER_HOME at an
// extracted upstream release (e.g.
// opentelemetry-dotnet-instrumentation-linux-glibc-<arch>.zip from
// github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases)
// instead of the Dash0 distro when running this test.
package efcorenet8_test

import (
	"context"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/harness"
	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestEntityFrameworkCorePostgresNet8(t *testing.T) {
	t.Skip("known upstream bug, not a regression -- see the package doc above and " +
		"test/e2e/README.md's \"Known failure: efcore\" section. Unskip to check " +
		"whether Npgsql 9.x's NpgsqlBatch tracing fix has been backported to the " +
		"8.x line EF Core 8 depends on.")

	sink := otelsink.Start(t)
	ctx := context.Background()

	nw := harness.NewNetwork(t, ctx)
	harness.StartBackingService(t, ctx, harness.BackingServiceOptions{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "postgres",
		},
		Network:    nw,
		Alias:      "postgres",
		WaitingFor: wait.ForExec([]string{"pg_isready", "-U", "postgres"}),
	})

	container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
		ExampleDir:  "examples/efcore-postgres-net8",
		TestdataDir: "test/e2e/testdata/efcore-postgres-net8",
		ExposedPort: "8080/tcp",
		WaitPath:    "/",
		Networks:    []string{nw},
	})

	status, body := harness.ContainerHTTPGet(t, ctx, container, "8080/tcp", "/query")
	require.Equal(t, 200, status, "unexpected response from /query: %s", body)

	traces := sink.WaitForTraces(t, 30*time.Second, func(tr *otelsink.Traces) bool {
		return tr.WithKind(tracepb.Span_SPAN_KIND_CLIENT).Len() > 0
	})

	serverSpans := traces.WithKind(tracepb.Span_SPAN_KIND_SERVER)
	assert.GreaterOrEqual(t, serverSpans.Len(), 1, "expected a server span for GET /query, got: %v", traces.Names())

	dbSpans := traces.WithKind(tracepb.Span_SPAN_KIND_CLIENT).WithSpanAttributeValue("db.system", "postgresql")
	assert.GreaterOrEqual(t, dbSpans.Len(), 1, "expected EntityFrameworkCore client spans with db.system=postgresql, got: %v", traces.Names())
}
