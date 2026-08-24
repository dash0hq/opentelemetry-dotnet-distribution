// SPDX-License-Identifier: Apache-2.0

// Package runtimemetrics validates OpenTelemetry.Instrumentation.Runtime and
// OpenTelemetry.Instrumentation.Process — both wired via compile-time
// extension method calls (no reflection), unlike the other contrib
// instrumentations, and enabled by default. Reuses the aspnetcore-httpclient
// scenario's app/Dockerfile since these instrumentations need no
// app-specific code, just any running .NET process.
package runtimemetrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/harness"
	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/stretchr/testify/assert"
)

func TestRuntimeAndProcessMetrics(t *testing.T) {
	sink := otelsink.Start(t)
	ctx := context.Background()

	container := harness.StartInstrumentedApp(t, ctx, sink, harness.AppScenario{
		ExampleDir:  "examples/aspnetcore-httpclient",
		TestdataDir: "test/e2e/testdata/aspnetcore-httpclient",
		ExposedPort: "8080/tcp",
		WaitPath:    "/",
	})

	// A little traffic to make sure the runtime has done some GC-relevant
	// work before the metrics collection interval elapses.
	for range 5 {
		harness.ContainerHTTPGet(t, ctx, container, "8080/tcp", "/")
	}

	// The default OTel metric export interval is 60s; give it enough room
	// past that rather than guessing at a shorter one.
	metrics := sink.WaitForMetrics(t, 90*time.Second, func(m *otelsink.Metrics) bool { return m.Len() > 0 })

	assert.GreaterOrEqual(t, metrics.WithName("process.runtime.dotnet.gc.collections.count").Len(), 1,
		"expected a Runtime instrumentation metric, got: %v", metrics.Names())
	assert.GreaterOrEqual(t, metrics.WithName("process.cpu.time").Len(), 1,
		"expected a Process instrumentation metric, got: %v", metrics.Names())
}
