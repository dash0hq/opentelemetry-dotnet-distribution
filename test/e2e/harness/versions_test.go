// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"fmt"
	"slices"
	"testing"
)

// fatalRecorder is a testing.TB whose Fatalf records instead of aborting, so
// the drift case can be asserted on rather than killing the test binary.
type fatalRecorder struct {
	testing.TB
	fatal string
}

func (f *fatalRecorder) Helper() {}

func (f *fatalRecorder) Fatalf(format string, args ...any) {
	f.fatal = fmt.Sprintf(format, args...)
}

func TestVersionsUnderTest(t *testing.T) {
	for _, tc := range []struct {
		name      string
		leg       string
		versions  []string
		want      []string
		wantFatal bool
	}{{
		name:     "unset leg runs the full sweep",
		leg:      "",
		versions: AllDotNetVersions,
		want:     AllDotNetVersions,
	}, {
		name:     "leg selects its own version",
		leg:      "9.0",
		versions: AllDotNetVersions,
		want:     []string{"9.0"},
	}, {
		name:     "leg a scenario does not target yields nothing",
		leg:      "6.0",
		versions: Net8PlusDotNetVersions,
		want:     nil,
	}, {
		name:     "leg within a narrowed sweep still selects",
		leg:      "10.0",
		versions: Net8PlusDotNetVersions,
		want:     []string{"10.0"},
	}, {
		name:      "unknown leg is a hard failure, not an empty pass",
		leg:       "11.0",
		versions:  AllDotNetVersions,
		wantFatal: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(LegVersionEnvVar, tc.leg)
			rec := &fatalRecorder{TB: t}

			got := VersionsUnderTest(rec, tc.versions)

			if tc.wantFatal {
				if rec.fatal == "" {
					t.Fatalf("expected a fatal for %s=%s, got none (returned %v)", LegVersionEnvVar, tc.leg, got)
				}
				return
			}
			if rec.fatal != "" {
				t.Fatalf("unexpected fatal: %s", rec.fatal)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("VersionsUnderTest(%v) with %s=%q = %v, want %v", tc.versions, LegVersionEnvVar, tc.leg, got, tc.want)
			}
		})
	}
}

// TestVersionsUnderTestCoversEveryLeg asserts every known version is covered
// by at least one scenario sweep, so no CI leg runs nothing at all.
func TestVersionsUnderTestCoversEveryLeg(t *testing.T) {
	sweeps := map[string][]string{
		"AllDotNetVersions":      AllDotNetVersions,
		"Net8PlusDotNetVersions": Net8PlusDotNetVersions,
	}
	for _, version := range AllDotNetVersions {
		covered := false
		for _, sweep := range sweeps {
			if slices.Contains(sweep, version) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("no scenario sweep covers .NET %s -- its CI leg would execute nothing", version)
		}
	}
}
