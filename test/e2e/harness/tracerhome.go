// SPDX-License-Identifier: Apache-2.0

// Package harness provides shared E2E test infrastructure: resolving the
// tracer-home under test and launching instrumented app containers wired to
// an otelsink.Sink via the real OTel injector.
package harness

import (
	"os"
	"testing"
)

// TracerHomeEnvVar names the environment variable TracerHome reads.
const TracerHomeEnvVar = "DASH0_E2E_TRACER_HOME"

// TracerHome returns the path to a ready-to-use tracer-home directory (the
// upstream bin/tracer-home layout: net/, AdditionalDeps/, linux-x64/, etc.),
// read from the DASH0_E2E_TRACER_HOME environment variable.
//
// Two things populate this directory before the test run, both producing the
// same layout so this code doesn't need to know which is in effect:
//
//   - Dev-loop mode: scripts/build-tracer-home-linux-x64.sh builds it from a
//     local source checkout of dash0hq/opentelemetry-dotnet-instrumentation,
//     to catch regressions before tagging a release there.
//   - Release-gate mode: the distribution's own release pipeline extracts
//     the tarballs it just built, to validate the exact artifacts about to
//     be published.
//
// Building (or downloading) the tracer-home is deliberately done once,
// outside the test binary, rather than memoized in Go: every test in a run
// shares the same directory simply by reading the same env var.
func TracerHome(t testing.TB) string {
	t.Helper()
	dir := os.Getenv(TracerHomeEnvVar)
	if dir == "" {
		t.Fatalf("%s is not set — build or extract a tracer-home first (see scripts/build-tracer-home-linux-x64.sh) and point this env var at it", TracerHomeEnvVar)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("%s=%s: %v", TracerHomeEnvVar, dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s=%s is not a directory", TracerHomeEnvVar, dir)
	}
	return dir
}
