// SPDX-License-Identifier: Apache-2.0

// Package aspnetcorenet8 is the net8.0 twin of the aspnetcore scenario. It
// exists specifically to catch instrumentation-image packaging regressions
// that only affect TFMs above the net6.0/net7.0 floor — e.g. the
// OpenTelemetry.Instrumentation.AspNetCore version pin in dash0-main's
// Directory.Packages.props is conditioned on TargetFramework, so a mistake
// there could silently break net8.0 tracing while net6.0 keeps working.
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

func TestAspNetCoreHttpClientNet8(t *testing.T) {
	sink := otelsink.Start(t)
	ctx := context.Background()

	container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
		ExampleDir:  "examples/aspnetcore-httpclient-net8",
		TestdataDir: "test/e2e/testdata/aspnetcore-httpclient-net8",
		ExposedPort: "8080/tcp",
		WaitPath:    "/",
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
	assert.True(t, strings.HasPrefix(runtimeVersion, "8."), "expected process.runtime.version to start with \"8.\", got %q", runtimeVersion)
}
