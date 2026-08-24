// SPDX-License-Identifier: Apache-2.0

// Package sqlclient exercises the Npgsql (PostgreSQL ADO.NET) instrumentation.
package sqlclient_test

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

func TestSqlClientPostgres(t *testing.T) {
	t.Skip("known regression: no ASP.NET Core server span is ever produced on " +
		"net6.0 -- see test/e2e/README.md's \"Known failure: ASP.NET Core " +
		"server spans on net6.0\" section. Unskip to check whether it's fixed " +
		"upstream.")

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
		ExampleDir:  "examples/sqlclient-postgres",
		TestdataDir: "test/e2e/testdata/sqlclient-postgres",
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

	// Npgsql's own ActivitySource instrumentation predates the OTel semconv
	// db.system -> db.system.name rename, unlike the StackExchange.Redis
	// instrumentation exercised by the rediscache scenario.
	dbSpans := traces.WithKind(tracepb.Span_SPAN_KIND_CLIENT).WithSpanAttribute("db.system")
	assert.GreaterOrEqual(t, dbSpans.Len(), 1, "expected Npgsql client spans carrying db.system, got: %v", traces.Names())

	postgresSpans := traces.WithSpanAttributeValue("db.system", "postgresql")
	assert.GreaterOrEqual(t, postgresSpans.Len(), 1, "expected db.system=postgresql on the Npgsql client spans, got: %v", traces.Names())
}
