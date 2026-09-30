// SPDX-License-Identifier: Apache-2.0

// Package grpc exercises OpenTelemetry.Instrumentation.GrpcNetClient. Its
// Dash0 initializer reflects into a public XxxInstrumentation type via a
// public constructor — the same shape already proven working (against the
// netstandard2.0 fallback net6.0 falls back to) by the rediscache
// scenario's StackExchangeRedis instrumentation.
package grpc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/harness"
	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// dotNetVersions are the TFMs examples/grpc-client multi-targets, matching
// what test/e2e/testdata/grpc-client's Dockerfile/Dockerfile.musl accept via
// their DASH0_DOTNET_VERSION build arg.
var dotNetVersions = harness.AllDotNetVersions

func TestGrpcNetClient(t *testing.T) {
	ctx := context.Background()

	for _, dotnetVersion := range harness.VersionsUnderTest(t, dotNetVersions) {
		t.Run(dotnetVersion, func(t *testing.T) {
			if dotnetVersion == "6.0" {
				t.Skip("known regression: no ASP.NET Core server span is ever produced on " +
					"net6.0 -- see test/e2e/README.md's \"Known failure: ASP.NET Core " +
					"server spans on net6.0\" section. Unskip to check whether it's fixed " +
					"upstream.")
			}

			sink := otelsink.Start(t)

			// See test/e2e/testdata/grpc-client/Dockerfile's own header: the jammy
			// base image is only needed at build time, and only for net6.0/net7.0,
			// to get a glibc new enough for Grpc.Tools' bundled arm64 protoc binary.
			sdkTagSuffix := ""
			if dotnetVersion == "6.0" || dotnetVersion == "7.0" {
				sdkTagSuffix = "-jammy"
			}

			container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
				ExampleDir:  "examples/grpc-client",
				TestdataDir: "test/e2e/testdata/grpc-client",
				ExposedPort: "8080/tcp",
				WaitPath:    "/",
				BuildArgs: map[string]string{
					"DASH0_DOTNET_VERSION": dotnetVersion,
					"SDK_TAG_SUFFIX":       sdkTagSuffix,
				},
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
