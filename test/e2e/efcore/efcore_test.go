// SPDX-License-Identifier: Apache-2.0

// Package efcore exercises OpenTelemetry.Instrumentation.EntityFrameworkCore
// (via the Npgsql EF Core provider).
//
// As of writing this test is a known, reproducible failure, not a
// test-authoring mistake, with two independent confirmed causes:
//
//  1. AspNetCoreInitializer bug (separate, unresolved). On net6.0, with both
//     ASP.NET Core hosting and EF Core active in the same process, it throws
//
//     System.IO.FileNotFoundException: Could not load file or assembly
//     'OpenTelemetry.Instrumentation.AspNetCore, Version=1.16.0.1140, ...'
//
//     even though the net6.0 tracer-home folder correctly ships the pinned
//     1.9.0.42 build (confirmed by inspecting the assembly's own metadata) —
//     the resolver's version-safety check (AssemblyResolver.Net.cs:
//     assemblyVersion < assemblyName.Version) rejects it because something
//     requests version 1.16.0.1140 specifically (matching the net8.0
//     folder's build) instead of an unversioned lookup. Doesn't reproduce in
//     sqlclient/rediscache/quartz/grpc, which also combine ASP.NET Core
//     hosting with another instrumentation successfully. This failure is
//     isolated per-initializer (see LazyInstrumentationLoader.cs) and does
//     not explain cause 2 below.
//
//  2. The actual reason no database span ever appears (root-caused, and not
//     specific to this .NET version or Npgsql version): EntityFrameworkCoreInitializer
//     deliberately suppresses EF Core's own span for the Npgsql provider
//     ("Configured EntityFrameworkCore instrumentation to skip Npgsql
//     provider because Npgsql instrumentation is enabled" — see
//     ConfigureNpgsqlSuppressionIfNeeded), on the assumption that Npgsql's
//     own native ActivitySource tracing (TracerInstrumentation.Npgsql =>
//     builder.AddSource("Npgsql")) will cover EF-Core-issued commands
//     instead. It never does: Npgsql's ActivitySource tracing
//     (NpgsqlActivitySource.CommandStart) is only called from
//     NpgsqlCommand's own execution path; EF Core's Npgsql provider executes
//     commands via NpgsqlBatch, which has never called into
//     NpgsqlActivitySource at all — confirmed absent in Npgsql v7.0.7
//     (the newest version resolvable for a net6.0/net7.0 app, since
//     EF Core 8's Npgsql provider requires net8.0) through v8.0.0 and
//     v9.0.0. A manually-added raw NpgsqlCommand query in the same request
//     alongside the EF Core call *did* produce a span, confirming Npgsql
//     tracing itself works fine in-process — only EF-Core-issued (batched)
//     commands are silently dropped.
//
// Net effect: EntityFrameworkCore + Npgsql produces zero database spans
// today, on any .NET/Npgsql version combination, until either Npgsql adds
// ActivitySource tracing to NpgsqlBatch, or the suppression is corrected to
// not defer to a source that will never fire.
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
	t.Skip("known upstream bug, not a regression -- see the package doc above and " +
		"test/e2e/README.md's \"Known failure: efcore\" section. Unskip to check " +
		"whether Npgsql has added NpgsqlBatch tracing or the EF Core suppression " +
		"logic has been fixed.")

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
