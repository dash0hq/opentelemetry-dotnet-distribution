// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"os"
	"slices"
	"testing"
)

// AllDotNetVersions is the full net6.0-net10.0 sweep every version-swept
// scenario (quartz, rediscache, sqlclient, grpc, ...) multi-targets and
// loops over via t.Run(dotnetVersion, ...). Shared here, rather than
// re-declared per package, so there's exactly one place to update when a
// new .NET version needs covering -- the other half of that update is
// build-and-e2e.yml's own dotnet-version matrix, which can't read this Go
// value: GitHub Actions has to know a matrix's legs before any job runs,
// so it necessarily keeps its own copy of this same list.
var AllDotNetVersions = []string{"6.0", "7.0", "8.0", "9.0", "10.0"}

// Net8PlusDotNetVersions is the net8.0-net10.0 subset the aspnetcorenet8/
// efcorenet8 twin scenarios multi-target -- see their package docs for why
// they don't cover net6.0/net7.0.
var Net8PlusDotNetVersions = AllDotNetVersions[2:]

// LegVersionEnvVar names the environment variable build-and-e2e.yml's
// per-version matrix sets to tell a leg which .NET version it owns.
const LegVersionEnvVar = "LEG_VERSION"

// VersionsUnderTest narrows versions -- the TFM list the calling scenario
// multi-targets -- to what this leg should exercise, and is what every
// version-swept scenario ranges over instead of its raw list.
//
// Unset LEG_VERSION (the local dev loop) means the full sweep. Set, it
// selects that one version, returning empty for a scenario that doesn't
// multi-target it: aspnetcorenet8/efcorenet8 on the 6.0 and 7.0 legs,
// legitimately. A leg AllDotNetVersions doesn't know about is a hard
// failure, since that can only mean the matrix and this list have drifted.
func VersionsUnderTest(t testing.TB, versions []string) []string {
	t.Helper()
	leg := os.Getenv(LegVersionEnvVar)
	if leg == "" {
		return versions
	}
	if !slices.Contains(AllDotNetVersions, leg) {
		t.Fatalf("%s=%s is not one of the known .NET versions %v -- build-and-e2e.yml's "+
			"dotnet-version matrix and harness.AllDotNetVersions have drifted apart",
			LegVersionEnvVar, leg, AllDotNetVersions)
	}
	if !slices.Contains(versions, leg) {
		return nil
	}
	return []string{leg}
}

// SkipUnlessLegVersion skips t unless this run's leg is version, for a
// scenario with no version sweep of its own that would otherwise repeat
// identically on every leg. LEG_VERSION unset always proceeds.
func SkipUnlessLegVersion(t testing.TB, version string) {
	t.Helper()
	leg := os.Getenv(LegVersionEnvVar)
	if leg != "" && leg != version {
		t.Skipf("only runs on the %s=%s leg", LegVersionEnvVar, version)
	}
}
