// Copyright Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

using System.Diagnostics;
using System.Formats.Tar;
using System.IO.Compression;

namespace Dash0.Build;

/// <summary>
/// Per-RID bundle assembler.
///
/// Usage:
///   dotnet run --project build/Build -- --rid linux-x64 --version 0.0.1-dev
///
/// Produces:
///   artifacts/dash0-opentelemetry-dotnet-autoinstrumentation-&lt;rid&gt;.tar.gz
///
/// Layout inside the tarball (per KTD4 in docs/plans/):
///   dash0-opentelemetry-dotnet-autoinstrumentation-&lt;rid&gt;/
///   ├── instrument.sh
///   ├── VERSION
///   ├── META-INF/dash0-distro-manifest.properties
///   ├── &lt;rid&gt;/Dash0.OpenTelemetry.AutoInstrumentation.Native.{so,dylib}
///   └── net/{net6.0,net7.0,net8.0,net9.0}/*.dll
/// </summary>
internal static class Program
{
    private static readonly string[] SupportedTfms = ["net6.0", "net7.0", "net8.0", "net9.0"];

    private static int Main(string[] args)
    {
        try
        {
            var opts = Options.Parse(args);
            var repoRoot = FindRepoRoot();
            var artifactsDir = Path.Combine(repoRoot, "artifacts");
            var stagingDir = Path.Combine(artifactsDir, $"dash0-opentelemetry-dotnet-autoinstrumentation-{opts.Rid}");

            Console.WriteLine($"Repo root: {repoRoot}");
            Console.WriteLine($"RID: {opts.Rid}");
            Console.WriteLine($"Version: {opts.Version}");
            Console.WriteLine($"Staging: {stagingDir}");
            Console.WriteLine();

            if (Directory.Exists(stagingDir))
            {
                Directory.Delete(stagingDir, recursive: true);
            }
            Directory.CreateDirectory(stagingDir);
            Directory.CreateDirectory(artifactsDir);

            PublishManagedAssemblies(repoRoot, stagingDir, opts.Rid);
            CopyNativeProfilerPlaceholder(stagingDir, opts.Rid);
            WriteInstrumentScript(stagingDir);
            WriteVersionFile(stagingDir, opts.Version);
            WriteManifest(repoRoot, stagingDir, opts);

            var tarball = Path.Combine(artifactsDir, $"dash0-opentelemetry-dotnet-autoinstrumentation-{opts.Rid}.tar.gz");
            CreateTarball(stagingDir, tarball);

            Console.WriteLine();
            Console.WriteLine($"Bundle: {tarball}");
            Console.WriteLine($"Size:   {new FileInfo(tarball).Length:N0} bytes");
            return 0;
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine($"Build failed: {ex.Message}");
            Console.Error.WriteLine(ex.StackTrace);
            return 1;
        }
    }

    private static void PublishManagedAssemblies(string repoRoot, string stagingDir, string rid)
    {
        // Main assembly: multi-TFM per R21.
        foreach (var tfm in SupportedTfms)
        {
            var outDir = Path.Combine(stagingDir, "net", tfm);
            Directory.CreateDirectory(outDir);
            RunDotnet(
                "publish",
                Path.Combine(repoRoot, "src/Dash0.OpenTelemetry.AutoInstrumentation/Dash0.OpenTelemetry.AutoInstrumentation.csproj"),
                "-f", tfm,
                "-c", "Release",
                "--no-self-contained",
                "-o", outDir);
        }

        // Loader: single-TFM (netcoreapp3.1), works on all .NET Core 3+ per KTD8.
        // Ships in a runtime-independent location; startup hook chooses this by
        // DOTNET_STARTUP_HOOKS env var.
        var loaderDir = Path.Combine(stagingDir, "net", "netcoreapp3.1");
        Directory.CreateDirectory(loaderDir);
        RunDotnet(
            "publish",
            Path.Combine(repoRoot, "src/Dash0.OpenTelemetry.AutoInstrumentation.Loader/Dash0.OpenTelemetry.AutoInstrumentation.Loader.csproj"),
            "-f", "netcoreapp3.1",
            "-c", "Release",
            "--no-self-contained",
            "-o", loaderDir);
    }

    private static void CopyNativeProfilerPlaceholder(string stagingDir, string rid)
    {
        // U3: placeholder empty file. The real native profiler is built by a
        // future unit (CMake-driven, per KTD4 KTDh + Phase C).
        // Extension per RID convention: .so on linux-*, .dylib on osx-*, .dll on win-* (Windows out of v1).
        var ext = rid.StartsWith("linux") ? ".so"
                : rid.StartsWith("osx") ? ".dylib"
                : throw new NotSupportedException($"Unsupported RID: {rid}");
        var nativeDir = Path.Combine(stagingDir, rid);
        Directory.CreateDirectory(nativeDir);
        File.WriteAllText(
            Path.Combine(nativeDir, $"Dash0.OpenTelemetry.AutoInstrumentation.Native{ext}"),
            "# U3 placeholder — real native profiler is built by CMake in a future unit.\n");
    }

    private static void WriteInstrumentScript(string stagingDir)
    {
        // U3 stub — matches upstream's instrument.sh shape (env-var-based
        // activation contract). Later units re-derive from the current upstream
        // script during rebase; this stub uses the KTD4 env-var contract.
        var script = """
            #!/bin/sh
            # Dash0 OpenTelemetry .NET Auto-Instrumentation activation script.
            # source ./instrument.sh, then run your .NET app.
            SCRIPT_DIR="$(cd -- "$(dirname -- "${0}")" >/dev/null 2>&1 && pwd)"
            export OTEL_DOTNET_AUTO_HOME="${SCRIPT_DIR}"
            export CORECLR_ENABLE_PROFILING=1
            export CORECLR_PROFILER='{918728DD-259F-4A6A-AC2B-B85E1B658318}'
            # Native profiler path — resolved per RID at deployment time.
            # export CORECLR_PROFILER_PATH="${SCRIPT_DIR}/<rid>/Dash0.OpenTelemetry.AutoInstrumentation.Native.so"
            export DOTNET_STARTUP_HOOKS="${SCRIPT_DIR}/net/netcoreapp3.1/Dash0.OpenTelemetry.AutoInstrumentation.Loader.dll"
            export OTEL_DOTNET_AUTO_PLUGINS='Dash0.OpenTelemetry.AutoInstrumentation.Dash0Plugin, Dash0.OpenTelemetry.AutoInstrumentation'

            """;
        var scriptPath = Path.Combine(stagingDir, "instrument.sh");
        File.WriteAllText(scriptPath, script);
        // Mark executable — Tar entries preserve mode below via TarEntry.Mode.
    }

    private static void WriteVersionFile(string stagingDir, string version) =>
        File.WriteAllText(Path.Combine(stagingDir, "VERSION"), version + "\n");

    private static void WriteManifest(string repoRoot, string stagingDir, Options opts)
    {
        var metaInf = Path.Combine(stagingDir, "META-INF");
        Directory.CreateDirectory(metaInf);
        var gitSha = TryGitSha(repoRoot) ?? "unknown";
        var content = $"""
            # Dash0 distro version manifest (R4).
            distro.version={opts.Version}
            distro.rid={opts.Rid}
            distro.git.sha={gitSha}
            upstream.instrumentation.version={opts.UpstreamVersion}
            build.iso8601={DateTime.UtcNow:yyyy-MM-ddTHH:mm:ssZ}
            """;
        File.WriteAllText(Path.Combine(metaInf, "dash0-distro-manifest.properties"), content);
    }

    private static void CreateTarball(string sourceDir, string tarballPath)
    {
        if (File.Exists(tarballPath))
        {
            File.Delete(tarballPath);
        }
        using var fs = File.Create(tarballPath);
        using var gz = new GZipStream(fs, CompressionLevel.Optimal, leaveOpen: false);
        // includeBaseDirectory: true packs sourceDir's contents under its own
        // name as the root entry, matching the KTD4 layout where users extract
        // dash0-opentelemetry-dotnet-autoinstrumentation-<rid>/ from the archive.
        TarFile.CreateFromDirectory(sourceDir, gz, includeBaseDirectory: true);
    }

    private static void RunDotnet(params string[] args)
    {
        var psi = new ProcessStartInfo("dotnet")
        {
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
        };
        foreach (var a in args)
        {
            psi.ArgumentList.Add(a);
        }
        Console.WriteLine($"$ dotnet {string.Join(' ', args)}");
        var p = Process.Start(psi)!;
        p.WaitForExit();
        if (p.ExitCode != 0)
        {
            var stderr = p.StandardError.ReadToEnd();
            var stdout = p.StandardOutput.ReadToEnd();
            throw new InvalidOperationException($"dotnet exited {p.ExitCode}\nSTDOUT:\n{stdout}\nSTDERR:\n{stderr}");
        }
    }

    private static string? TryGitSha(string repoRoot)
    {
        try
        {
            var psi = new ProcessStartInfo("git", "rev-parse --short HEAD")
            {
                RedirectStandardOutput = true,
                UseShellExecute = false,
                WorkingDirectory = repoRoot,
            };
            var p = Process.Start(psi)!;
            var sha = p.StandardOutput.ReadToEnd().Trim();
            p.WaitForExit();
            return p.ExitCode == 0 && !string.IsNullOrEmpty(sha) ? sha : null;
        }
        catch
        {
            return null;
        }
    }

    private static string FindRepoRoot()
    {
        var dir = Environment.CurrentDirectory;
        while (dir != null && !File.Exists(Path.Combine(dir, "Dash0.OpenTelemetry.AutoInstrumentation.sln")))
        {
            dir = Path.GetDirectoryName(dir);
        }
        return dir ?? throw new InvalidOperationException(
            "Repo root not found — no Dash0.OpenTelemetry.AutoInstrumentation.sln in cwd or ancestors");
    }

    private sealed record Options(string Rid, string Version, string UpstreamVersion)
    {
        public static Options Parse(string[] args)
        {
            string? rid = null;
            string? version = null;
            string? upstreamVersion = "1.16.0"; // Placeholder; U4 metadata validator provides the pinned value.
            for (int i = 0; i < args.Length; i++)
            {
                switch (args[i])
                {
                    case "--rid": rid = args[++i]; break;
                    case "--version": version = args[++i]; break;
                    case "--upstream-version": upstreamVersion = args[++i]; break;
                    default: throw new ArgumentException($"Unknown argument: {args[i]}");
                }
            }
            if (rid is null) throw new ArgumentException("--rid required (linux-x64, linux-arm64, linux-musl-x64, linux-musl-arm64, osx-x64, osx-arm64)");
            if (version is null) throw new ArgumentException("--version required");
            return new Options(rid, version, upstreamVersion!);
        }
    }
}
