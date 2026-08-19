# E2E tests

Builds an example app (see `../../examples`) with the real OTel injector
loaded via `LD_PRELOAD` and a tracer-home under test wired in, then asserts
against real telemetry received by an in-process OTLP sink
([`otelsink`](otelsink/)) — proving actual auto-instrumentation behavior end
to end, not just that the tracer-home builds.

## Running

Point `DASH0_E2E_TRACER_HOME` at a directory laid out like the upstream
`bin/tracer-home` (`net/`, `linux-<arch>/`, `instrument.sh`, ...) — either:

- **Dev-loop**: build it from a local source checkout of
  `dash0hq/opentelemetry-dotnet-instrumentation`, to catch regressions
  before tagging a release there:
  ```
  ../../scripts/build-tracer-home-linux-arm64.sh /path/to/opentelemetry-dotnet-instrumentation
  # or build-tracer-home-linux-x64.sh, if that's the arch you need
  export DASH0_E2E_TRACER_HOME=/path/to/opentelemetry-dotnet-instrumentation/bin/tracer-home
  ```
  Prefer the script matching your machine's native architecture — building
  the x64 native library under QEMU emulation on Apple Silicon has proven
  unreliable (observed `dpkg` and `tar` crashing mid-build under emulation).
  Real CI runs natively on both amd64 and arm64 runners either way.

- **Release-gate**: extract the tarball(s) this distribution's own release
  pipeline just built, to validate the exact artifacts about to be
  published:
  ```
  mkdir tracer-home && tar -C tracer-home -xzf dash0-opentelemetry-dotnet-instrumentation-linux-x64.tar.gz
  export DASH0_E2E_TRACER_HOME=$PWD/tracer-home
  ```

Then, from `test/e2e`:

```
go test ./...
```

Requires Docker. On Colima, `DOCKER_HOST` and
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` may need to be set explicitly —
testcontainers-go doesn't always pick up the active Colima context, and the
value to bind-mount for its sidecar containers (the reaper, and the sshd
container backing `HostAccessPorts`) is the socket path *inside* the Docker
daemon's own environment, not the host-side `DOCKER_HOST` value:

```
export DOCKER_HOST="unix://$HOME/.colima/default/docker.sock"
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
```

Scenarios build and run for the host's own architecture (`runtime.GOARCH`)
rather than a fixed platform, so both local dev and CI run natively instead
of under emulation. If a build inexplicably targets the wrong architecture,
a base image pulled natively before switching machines (or before this pin
existed) may be cached locally under the same tag; `docker rmi` it to force
a re-pull for the correct platform.

## Layout

- `otelsink/` — an in-process OTLP sink, copied from
  `open-telemetry/opentelemetry-packaging`'s `testutil/otelsink` (see
  `NOTICE`). Not modified beyond the import path.
- `harness/` — resolves the tracer-home under test (`TracerHome`), builds and
  starts scenario containers wired to a sink (`StartInstrumentedApp`), and
  starts plain backing-service containers for scenarios with a dependency
  (`StartBackingService`, `NewNetwork`).
- `testdata/<scenario>/` — each scenario's `Dockerfile` (layers the injector
  and tracer-home onto an example app) and any files it `COPY`s in (e.g.
  `injector.conf`).
- `<scenario>/` (e.g. `aspnetcore/`) — the actual test files.

## Scenarios

| Package            | Example app                       | TFM  | Instrumentation exercised                  |
| ------------------ | ---------------------------------- | ---- | ------------------------------------------- |
| `aspnetcore`        | `aspnetcore-httpclient`            | net6.0 | ASP.NET Core (server) + HttpClient (client) |
| `aspnetcorenet8`    | `aspnetcore-httpclient-net8`       | net8.0 | Same, twin scenario for the net8.0+ pin path |
| `sqlclient`         | `sqlclient-postgres`               | net6.0 | Npgsql (ADO.NET client spans against a real Postgres backing container) |
| `rediscache`        | `redis-cache`                      | net6.0 | StackExchange.Redis (client spans against a real Redis backing container) |

A caveat worth knowing before writing new assertions: Npgsql's and
StackExchange.Redis's own instrumentation both still tag spans with the
older `db.system` attribute key, not the newer `db.system.name` — check the
actual span (e.g. by temporarily logging `sv.Span.GetAttributes()`) rather
than assuming a semconv version.

## Adding a scenario

1. Add (or reuse) an example app under `../../examples`.
2. Add `testdata/<scenario>/Dockerfile` and `injector.conf` — see
   `testdata/aspnetcore-httpclient` for the pattern. The tracer-home is
   staged at `tracer-home/glibc/` in the build context (matching how
   `dash0-operator`'s `download-instrumentation.sh` lays out each libc
   flavor side by side under one path-prefix directory, which is what the
   injector's `dotnet_auto_instrumentation_agent_path_prefix` expects to
   find `glibc/linux-<arch>/...` or `musl/linux-<arch>/...` under).
3. If the app needs a backing service (database, cache, ...), create a
   shared network with `harness.NewNetwork` and start the dependency with
   `harness.StartBackingService`, aliased to whatever hostname the app's own
   connection-string default expects (see `sqlclient/sqlclient_test.go` for
   the pattern) — pass the same network name in `AppScenario.Networks`.
4. Write the test using `harness.StartInstrumentedApp` and `otelsink`'s
   query/wait helpers — see `aspnetcore/aspnetcore_test.go`.
