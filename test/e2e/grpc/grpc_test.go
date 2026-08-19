// SPDX-License-Identifier: Apache-2.0

// Package grpc exercises OpenTelemetry.Instrumentation.GrpcNetClient. Its
// Dash0 initializer reflects into a public XxxInstrumentation type via a
// public constructor — the same shape already proven working (against the
// netstandard2.0 fallback net6.0 falls back to) by the rediscache
// scenario's StackExchangeRedis instrumentation.
package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/harness"
	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestGrpcNetClient(t *testing.T) {
	sink := otelsink.Start(t)
	ctx := context.Background()

	container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
		ExampleDir:  "examples/grpc-client",
		TestdataDir: "test/e2e/testdata/grpc-client",
		ExposedPort: "8080/tcp",
		WaitPath:    "/",
	})

	status, body := harness.ContainerHTTPGet(t, ctx, container, "8080/tcp", "/call")
	require.Equal(t, 200, status, "unexpected response from /call: %s", body)

	traces := sink.WaitForTraces(t, 30*time.Second, func(tr *otelsink.Traces) bool {
		return tr.WithName("greet.Greeter/SayHello").Len() > 0
	})

	serverSpans := traces.WithKind(tracepb.Span_SPAN_KIND_SERVER)
	assert.GreaterOrEqual(t, serverSpans.Len(), 1, "expected a server span for GET /call, got: %v", traces.Names())

	grpcSpans := traces.WithName("greet.Greeter/SayHello").WithKind(tracepb.Span_SPAN_KIND_CLIENT)
	assert.GreaterOrEqual(t, grpcSpans.Len(), 1, "expected a Grpc.Net.Client span for the SayHello call, got: %v", traces.Names())
	assert.Equal(t, 1, traces.WithSpanAttributeValue("rpc.system.name", "grpc").Len(),
		"expected the gRPC client span to carry rpc.system.name=grpc")
}
