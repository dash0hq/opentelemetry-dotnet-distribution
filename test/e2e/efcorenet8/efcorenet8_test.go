// SPDX-License-Identifier: Apache-2.0

// Package efcorenet8 is the net8.0+ counterpart of the efcore scenario. It
// exists to check whether the missing-database-span issue found there is
// specific to the Dash0 distro / net6.0, or reproduces with any
// OTEL_DOTNET_AUTO_HOME-compatible tracer-home (including the actual
// upstream open-telemetry/opentelemetry-dotnet-instrumentation release) on
// a modern .NET version with a current EF Core/Npgsql pairing.
//
// Multi-targets net8.0-net10.0 (one subtest per TFM), unlike the efcore
// scenario which stays pinned to net6.0 -- see test/e2e/README.md's
// Scenarios table. All three subtests are skipped identically:
// EfCorePostgresNet8.csproj pins Npgsql.EntityFrameworkCore.PostgreSQL to a
// single fixed version (8.0.11) across every TFM, same as every other
// multi-targeted scenario here, so which .NET runtime executes it doesn't
// change which Npgsql provider code actually runs -- unlike the net6.0-only
// ASP.NET-Core-server-span failure (see README.md), this bug is tied to the
// pinned package version, not the runtime, so there's no reason to expect
// net9.0/net10.0 to behave any differently from net8.0 unless that package
// pin itself changes.
//
// To check against upstream specifically, point DASH0_E2E_TRACER_HOME at an
// extracted upstream release (e.g.
// opentelemetry-dotnet-instrumentation-linux-glibc-<arch>.zip from
// github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases)
// instead of the Dash0 distro when running this test.
package efcorenet8_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/harness"
	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// dotNetVersions are the TFMs examples/efcore-postgres-net8 multi-targets,
// matching what test/e2e/testdata/efcore-postgres-net8's Dockerfile/
// Dockerfile.musl accept via their DASH0_DOTNET_VERSION build arg.
var dotNetVersions = harness.Net8PlusDotNetVersions

func TestEntityFrameworkCorePostgresNet8(t *testing.T) {
	ctx := context.Background()

	// One backing service shared by every version subtest below -- it's just
	// a Postgres instance, unrelated to which .NET version the app under
	// test runs, so there's no reason to pay for three of them.
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

	for _, dotnetVersion := range harness.VersionsUnderTest(t, dotNetVersions) {
		t.Run(dotnetVersion, func(t *testing.T) {
			t.Skip("known upstream bug, not a regression -- see the package doc above and " +
				"test/e2e/README.md's \"Known failure: efcore\" section. Unskip to check " +
				"whether Npgsql 9.x's NpgsqlBatch tracing fix has been backported to the " +
				"8.x line EF Core 8 depends on.")

			sink := otelsink.Start(t)

			container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
				ExampleDir:  "examples/efcore-postgres-net8",
				TestdataDir: "test/e2e/testdata/efcore-postgres-net8",
				ExposedPort: "8080/tcp",
				WaitPath:    "/",
				Networks:    []string{nw},
				BuildArgs:   map[string]string{"DASH0_DOTNET_VERSION": dotnetVersion},
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

			require.NotZero(t, traces.Len())
			runtimeVersion := ""
			for _, kv := range traces.Spans()[0].Resource.GetAttributes() {
				if kv.GetKey() == "process.runtime.version" {
					runtimeVersion = otelsink.AttrString(kv.GetValue())
				}
			}
			assert.True(t, strings.HasPrefix(runtimeVersion, dotnetVersion+"."),
				"expected process.runtime.version to start with %q, got %q", dotnetVersion+".", runtimeVersion)
		})
	}
}
