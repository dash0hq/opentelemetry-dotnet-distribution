// SPDX-License-Identifier: Apache-2.0

// Package quartz exercises OpenTelemetry.Instrumentation.Quartz. Its Dash0
// initializer reflects into a public XxxInstrumentation type via a public
// constructor — the same shape already proven working (against the
// netstandard2.0 fallback net6.0 falls back to) by the rediscache
// scenario's StackExchangeRedis instrumentation.
package quartz_test

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

// dotNetVersions are the TFMs examples/quartz-job multi-targets, matching
// what test/e2e/testdata/quartz-job's Dockerfile/Dockerfile.musl accept via
// their DASH0_DOTNET_VERSION build arg.
var dotNetVersions = harness.AllDotNetVersions

func TestQuartzJob(t *testing.T) {
	for _, dotnetVersion := range dotNetVersions {
		t.Run(dotnetVersion, func(t *testing.T) {
			if dotnetVersion == "6.0" {
				t.Skip("known regression: no ASP.NET Core server span is ever produced on " +
					"net6.0 -- see test/e2e/README.md's \"Known failure: ASP.NET Core " +
					"server spans on net6.0\" section. Unskip to check whether it's fixed " +
					"upstream.")
			}

			sink := otelsink.Start(t)
			ctx := context.Background()

			harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
				ExampleDir:  "examples/quartz-job",
				TestdataDir: "test/e2e/testdata/quartz-job",
				ExposedPort: "8080/tcp",
				WaitPath:    "/",
				BuildArgs:   map[string]string{"DASH0_DOTNET_VERSION": dotnetVersion},
			})

			// The job fires every 2 seconds starting on app startup; no HTTP call
			// needed to trigger it.
			traces := sink.WaitForTraces(t, 30*time.Second, func(tr *otelsink.Traces) bool {
				return tr.WithName("execute example-job").Len() > 0
			})

			jobSpans := traces.WithName("execute example-job")
			assert.GreaterOrEqual(t, jobSpans.Len(), 1, "expected Quartz job execution spans, got: %v", traces.Names())
			assert.GreaterOrEqual(t, traces.WithSpanAttributeValue("job.name", "example-job").WithName("execute example-job").Len(), 1,
				"expected the job execution span(s) to carry job.name=example-job")

			// AspNetCore instrumentation working here too is itself informative: it
			// confirms the version-resolution failure found in the efcore scenario
			// isn't simply "AspNetCore plus any other instrumentation" — Quartz
			// doesn't trigger it.
			serverSpans := traces.WithKind(tracepb.Span_SPAN_KIND_SERVER)
			assert.GreaterOrEqual(t, serverSpans.Len(), 1, "expected the readiness probe's server span, got: %v", traces.Names())

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
