// SPDX-License-Identifier: Apache-2.0

// Package aspnetcorenet8 is the net8.0+ twin of the aspnetcore scenario. It
// exists specifically to catch instrumentation-image packaging regressions
// that only affect TFMs above the net6.0/net7.0 floor — e.g. the
// OpenTelemetry.Instrumentation.AspNetCore version pin in dash0-main's
// Directory.Packages.props is conditioned on TargetFramework, so a mistake
// there could silently break net8.0+ tracing while net6.0 keeps working.
// Multi-targets net8.0-net10.0 (one subtest per TFM), unlike the aspnetcore
// scenario which stays pinned to net6.0 -- see test/e2e/README.md's
// Scenarios table.
package aspnetcorenet8_test

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

// dotNetVersions are the TFMs examples/aspnetcore-httpclient-net8
// multi-targets, matching what test/e2e/testdata/aspnetcore-httpclient-net8's
// Dockerfile/Dockerfile.musl accept via their DASH0_DOTNET_VERSION build arg.
var dotNetVersions = harness.Net8PlusDotNetVersions

func TestAspNetCoreHttpClientNet8(t *testing.T) {
	ctx := context.Background()

	for _, dotnetVersion := range harness.VersionsUnderTest(t, dotNetVersions) {
		t.Run(dotnetVersion, func(t *testing.T) {
			sink := otelsink.Start(t)

			container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
				ExampleDir:  "examples/aspnetcore-httpclient-net8",
				TestdataDir: "test/e2e/testdata/aspnetcore-httpclient-net8",
				ExposedPort: "8080/tcp",
				WaitPath:    "/",
				BuildArgs:   map[string]string{"DASH0_DOTNET_VERSION": dotnetVersion},
			})

			status, body := harness.ContainerHTTPGet(t, ctx, container, "8080/tcp", "/call")
			require.Equal(t, 200, status, "unexpected response from /call: %s", body)

			traces := sink.WaitForTraces(t, 30*time.Second, func(tr *otelsink.Traces) bool { return tr.Len() >= 2 })

			serverSpans := traces.WithKind(tracepb.Span_SPAN_KIND_SERVER)
			assert.GreaterOrEqual(t, serverSpans.Len(), 2, "expected server spans for both /call and /downstream, got: %v", traces.Names())

			clientSpans := traces.WithKind(tracepb.Span_SPAN_KIND_CLIENT)
			assert.GreaterOrEqual(t, clientSpans.Len(), 1, "expected an HttpClient client span for the outbound call to /downstream, got: %v", traces.Names())

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
