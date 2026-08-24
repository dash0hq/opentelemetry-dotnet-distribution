// SPDX-License-Identifier: Apache-2.0

// Package rediscache exercises the StackExchange.Redis instrumentation.
package rediscache_test

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

func TestRedisCache(t *testing.T) {
	t.Skip("known regression: no ASP.NET Core server span is ever produced on " +
		"net6.0 -- see test/e2e/README.md's \"Known failure: ASP.NET Core " +
		"server spans on net6.0\" section. Unskip to check whether it's fixed " +
		"upstream.")

	sink := otelsink.Start(t)
	ctx := context.Background()

	nw := harness.NewNetwork(t, ctx)
	harness.StartBackingService(t, ctx, harness.BackingServiceOptions{
		Image:      "redis:7-alpine",
		Network:    nw,
		Alias:      "redis",
		WaitingFor: wait.ForExec([]string{"redis-cli", "ping"}),
	})

	container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
		ExampleDir:  "examples/redis-cache",
		TestdataDir: "test/e2e/testdata/redis-cache",
		ExposedPort: "8080/tcp",
		WaitPath:    "/",
		Networks:    []string{nw},
	})

	status, body := harness.ContainerHTTPGet(t, ctx, container, "8080/tcp", "/cache")
	require.Equal(t, 200, status, "unexpected response from /cache: %s", body)

	traces := sink.WaitForTraces(t, 30*time.Second, func(tr *otelsink.Traces) bool {
		return tr.WithKind(tracepb.Span_SPAN_KIND_CLIENT).Len() > 0
	})

	serverSpans := traces.WithKind(tracepb.Span_SPAN_KIND_SERVER)
	assert.GreaterOrEqual(t, serverSpans.Len(), 1, "expected a server span for GET /cache, got: %v", traces.Names())

	// The StackExchange.Redis instrumentation also predates the OTel semconv
	// db.system -> db.system.name rename (Dash0's backend normalizes this on
	// ingestion, but the raw span still carries the older key).
	redisSpans := traces.WithSpanAttributeValue("db.system", "redis")
	assert.GreaterOrEqual(t, redisSpans.Len(), 1, "expected db.system=redis on the StackExchange.Redis client spans, got: %v", traces.Names())
}
