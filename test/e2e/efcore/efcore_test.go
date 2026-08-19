// SPDX-License-Identifier: Apache-2.0

// Package efcore exercises OpenTelemetry.Instrumentation.EntityFrameworkCore
// (via the Npgsql EF Core provider).
//
// As of writing this test is a known, reproducible failure, not a
// test-authoring mistake: on net6.0, with both ASP.NET Core hosting and EF
// Core active in the same process, AspNetCoreInitializer throws
//
//	System.IO.FileNotFoundException: Could not load file or assembly
//	'OpenTelemetry.Instrumentation.AspNetCore, Version=1.16.0.1140, ...'
//
// even though the net6.0 tracer-home folder correctly ships the pinned
// 1.9.0.42 build (confirmed by inspecting the assembly's own metadata) —
// the assembly resolver's version-safety check
// (AssemblyResolver.Net.cs: assemblyVersion < assemblyName.Version) rejects
// it because something requests version 1.16.0.1140 specifically (matching
// the net8.0 folder's build) instead of an unversioned lookup. No client
// span (from EF Core or the underlying Npgsql native ActivitySource) is
// produced either. This does not reproduce in the sqlclient/rediscache
// scenarios, which also combine ASP.NET Core hosting with another
// instrumentation successfully — so it is not simply "AspNetCore plus
// anything else breaks." Left as a real, currently-failing regression test
// rather than adjusted to match the broken behavior.
package efcore_test

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

func TestEntityFrameworkCorePostgres(t *testing.T) {
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
		ExampleDir:  "examples/efcore-postgres",
		TestdataDir: "test/e2e/testdata/efcore-postgres",
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

	// EntityFrameworkCore's instrumentation tags the standard db.* semconv
	// attributes on its client spans, same as the underlying Npgsql ADO
	// spans exercised by the sqlclient scenario.
	dbSpans := traces.WithKind(tracepb.Span_SPAN_KIND_CLIENT).WithSpanAttributeValue("db.system", "postgresql")
	assert.GreaterOrEqual(t, dbSpans.Len(), 1, "expected EntityFrameworkCore client spans with db.system=postgresql, got: %v", traces.Names())
}
