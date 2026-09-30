// SPDX-License-Identifier: Apache-2.0

package harness

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
