// Copyright Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

// The DOTNET_STARTUP_HOOKS contract requires a top-level type named StartupHook
// in the global namespace with a public static void Initialize() method.
// See https://github.com/dotnet/runtime/blob/main/docs/design/features/host-startup-hook.md

/// <summary>
/// Dash0 startup hook entry point. Runs before the target application's Main
/// method. In U3 this is a no-op stub; U6 wires the activation guard here,
/// U7 registers the flush-on-shutdown handler, U8 loads resource detectors.
/// </summary>
internal static class StartupHook
{
    public static void Initialize()
    {
        // U3 stub — populated in U6/U7/U8.
    }
}
