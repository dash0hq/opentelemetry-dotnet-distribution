// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/dash0hq/opentelemetry-dotnet-distribution/test/e2e/otelsink"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// buildArch is the container architecture scenarios build and run under:
// the host's own (runtime.GOARCH), so both local dev and CI run natively
// rather than under QEMU emulation. Go's "amd64"/"arm64" values match
// Docker's platform architecture names directly.
//
// Emulation matters here beyond the usual slowness: the injector reads its
// own process's ELF header from /proc/self/exe to detect the runtime, and
// that read fails under qemu-user binfmt translation (confirmed by running
// an amd64 container under emulation on an arm64 dev machine — the injector
// panicked on "Cannot read ELF header from /proc/self/exe"). Scenarios must
// have a tracer-home and injector binary available for whichever
// architecture they actually run on; there is currently only one scenario,
// wired for glibc x64/arm64.
var buildArch = runtime.GOARCH

// AppScenario describes an example app to build with the injector and the
// tracer-home under test wired in.
type AppScenario struct {
	// ExampleDir is the example app's directory, relative to the repo root
	// (e.g. "examples/aspnetcore-httpclient"). Staged into the build
	// context under app/.
	ExampleDir string
	// TestdataDir is the scenario's own directory, relative to the repo
	// root (e.g. "test/e2e/testdata/aspnetcore-httpclient"). Must contain a
	// Dockerfile that layers the injector and tracer-home on top of the
	// staged app/ and tracer-home/ directories; may contain other files
	// (e.g. injector.conf) that Dockerfile COPYs in. Staged at the build
	// context root.
	TestdataDir string
	// ExposedPort is the container port the app listens on, e.g. "8080/tcp".
	ExposedPort string
	// WaitPath is an HTTP path on ExposedPort that returns 2xx once the app
	// is ready.
	WaitPath string
	// Networks are Docker network names to join, for scenarios with a
	// backing service (see StartBackingService) the app needs to reach by
	// its network alias.
	Networks []string
	// NetworkAliases are the app container's own aliases on each of
	// Networks, keyed by network name. Usually unneeded — the app is the
	// caller, not something a backing service needs to resolve by name.
	NetworkAliases map[string][]string
}

// StartInstrumentedApp builds and starts scenario's app container with the
// real OTel injector loaded via LD_PRELOAD and pointed at the tracer-home
// under test (harness.TracerHome), wired to sink via HostAccessPorts so the
// injected instrumentation's OTLP exports reach it.
func StartInstrumentedApp(t testing.TB, ctx context.Context, sink *otelsink.Sink, scenario AppScenario) testcontainers.Container {
	t.Helper()
	root := repoRoot(t)
	tracerHome := TracerHome(t)
	buildContext := stageBuildContext(t, root, scenario, tracerHome)

	env := map[string]string{
		// The .NET tracer's own diagnostic logs default to a file inside
		// the container (OTEL_DOTNET_AUTO_LOGGER default: "file"), which
		// this harness never extracts -- so an instrumentation-side
		// failure (e.g. an initializer exception) is invisible even with
		// the container's own stdout/stderr captured on failure. Route it
		// to console instead, at debug verbosity, so it lands in the log
		// capture in the same place as the app's own output and the
		// injector's debug log.
		"OTEL_DOTNET_AUTO_LOGGER": "console",
		"OTEL_LOG_LEVEL":          "debug",
	}
	maps.Copy(env, sink.Env())

	injectorArch := buildArch
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       buildContext,
			Dockerfile:    "Dockerfile",
			KeepImage:     true,
			PrintBuildLog: true,
			BuildArgs:     map[string]*string{"INJECTOR_ARCH": &injectorArch},
			BuildOptionsModifier: func(opts *client.ImageBuildOptions) {
				opts.Platforms = []ocispec.Platform{{OS: "linux", Architecture: buildArch}}
			},
		},
		ImagePlatform:   "linux/" + buildArch,
		ExposedPorts:    []string{scenario.ExposedPort},
		Env:             env,
		HostAccessPorts: sink.HostAccessPorts(),
		Networks:        scenario.Networks,
		NetworkAliases:  scenario.NetworkAliases,
		WaitingFor:      wait.ForHTTP(scenario.WaitPath).WithPort(scenario.ExposedPort).WithStartupTimeout(2 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start %s", scenario.ExampleDir)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})
	// Registered after the Terminate cleanup above, so it runs first
	// (t.Cleanup is LIFO) and reads the container's log buffer before the
	// container itself goes away. Only on failure -- this is the injector's
	// (OTEL_INJECTOR_LOG_LEVEL=debug) and the app's own stdout/stderr, which
	// is verbose and only useful when a scenario didn't produce the traces
	// it was expected to.
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		logs, err := container.Logs(context.Background())
		if err != nil {
			t.Logf("could not fetch container logs for %s: %v", scenario.ExampleDir, err)
			return
		}
		defer logs.Close()
		data, _ := io.ReadAll(logs)
		t.Logf("--- container logs (%s) ---\n%s\n--- end container logs ---", scenario.ExampleDir, data)
	})

	return container
}

// ContainerHTTPGet sends an HTTP GET to path on container's mapped port and
// returns the response status code and body.
func ContainerHTTPGet(t testing.TB, ctx context.Context, container testcontainers.Container, port, path string) (int, string) {
	t.Helper()
	host, err := container.Host(ctx)
	require.NoError(t, err)
	mapped, err := container.MappedPort(ctx, port)
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s:%s%s", host, mapped.Port(), path)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

// repoRoot returns the absolute path to the opentelemetry-dotnet-distribution
// repo root, found by walking up from the working directory to the nearest
// ancestor containing .git. Unlike a go.mod search, this repo's outer tree
// (the .NET distribution) has no go.mod of its own — only this module,
// nested under test/e2e, does — so a go.mod search would stop one level
// too shallow.
func repoRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not find repo root (no .git found above %s)", dir)
		dir = parent
	}
}

// stageBuildContext assembles a temp directory combining the scenario's
// Dockerfile (and sibling files, e.g. injector.conf), the example app's
// source, and the tracer-home under test, since Docker's classic builder
// needs everything COPY references under one context directory.
func stageBuildContext(t testing.TB, root string, scenario AppScenario, tracerHome string) string {
	t.Helper()
	ctxDir := t.TempDir()

	copyTree(t, filepath.Join(root, scenario.TestdataDir), ctxDir)
	copyTree(t, filepath.Join(root, scenario.ExampleDir), filepath.Join(ctxDir, "app"))
	// Nested under glibc/, matching how dash0-operator's own
	// download-instrumentation.sh lays out each libc flavor's extracted
	// tarball (glibc/, musl/) side by side under one path-prefix directory:
	// the injector auto-detects the running process's libc and looks under
	// "<prefix>/glibc/linux-<arch>/..." or ".../musl/...", not the tarball's
	// own top-level linux-<arch>/ directly.
	copyTree(t, tracerHome, filepath.Join(ctxDir, "tracer-home", "glibc"))

	return ctxDir
}

// copyTree copies src's contents into dst (created if needed), skipping
// .NET build output directories that shouldn't be baked into the image.
func copyTree(t testing.TB, src, dst string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dst, 0o755))
	// #nosec G204 -- src/dst are test-controlled paths (repo tree and
	// TracerHome), not user input.
	cmd := exec.Command("cp", "-a", src+"/.", dst+"/")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cp -a %s -> %s: %s", src, dst, out)
	for _, junk := range []string{"bin", "obj"} {
		_ = os.RemoveAll(filepath.Join(dst, junk))
	}
}
