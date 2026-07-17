// Copyright Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

namespace Dash0.OpenTelemetry.AutoInstrumentation;

/// <summary>
/// Entry point registered via <c>OTEL_DOTNET_AUTO_PLUGINS</c>. Runs before other plugins
/// so the activation guard (U6) can short-circuit registration before any Dash0 plugin
/// modules load, resource detectors run, or the shutdown handler is installed.
/// </summary>
/// <remarks>
/// U3 stub — the guard, plugin registration, and shutdown handler are implemented in
/// U6, U7, U8. This class exists so the bundle has a well-known type to reference from
/// <c>OTEL_DOTNET_AUTO_PLUGINS=Dash0.OpenTelemetry.AutoInstrumentation.Dash0Plugin</c>.
/// </remarks>
public sealed class Dash0Plugin
{
    // Intentionally empty in U3 — populated in U6/U7/U8.
}
