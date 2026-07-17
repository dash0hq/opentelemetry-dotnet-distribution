using System;
using System.Reflection;

namespace Dash0.Spike.Consumer;

internal static class Program
{
    private const string ExpectedDash0DistroValue = "dash0-opentelemetry-dotnet-instrumentation";
    private const string UpstreamDistroValue = "opentelemetry-dotnet-instrumentation";
    private const string ExpectedCaseAMarker = "u1-case-a-marker";

    private static int Main()
    {
        // Load the OpenTelemetry.AutoInstrumentation assembly. Under substitution it comes from
        // the fork-checkout project build output; without substitution it comes from the NuGet cache.
        var asm = Assembly.Load("OpenTelemetry.AutoInstrumentation");
        Console.WriteLine($"Loaded assembly: {asm.FullName}");
        Console.WriteLine($"Location: {asm.Location}");

        // Constants is internal — read via reflection.
        var constantsType = asm.GetType("OpenTelemetry.AutoInstrumentation.Constants", throwOnError: true);
        var distAttrsType = constantsType!.GetNestedType("DistributionAttributes", BindingFlags.Public | BindingFlags.NonPublic)
            ?? throw new InvalidOperationException("Nested type DistributionAttributes not found");

        var distroNameField = distAttrsType.GetField(
            "TelemetryDistroNameAttributeValue",
            BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.Static)
            ?? throw new InvalidOperationException("TelemetryDistroNameAttributeValue field not found");
        var distroNameValue = (string?)distroNameField.GetRawConstantValue();

        var markerField = distAttrsType.GetField(
            "Dash0SpikeMarker",
            BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.Static);
        var markerValue = markerField is null ? null : (string?)markerField.GetRawConstantValue();

        Console.WriteLine();
        Console.WriteLine("=== Case A (additive: Dash0SpikeMarker) ===");
        Console.WriteLine($"  Field present: {markerField is not null}");
        Console.WriteLine($"  Value: {markerValue ?? "<null>"}");
        Console.WriteLine($"  Expected: {ExpectedCaseAMarker}");

        Console.WriteLine();
        Console.WriteLine("=== Case B (modified: TelemetryDistroNameAttributeValue) ===");
        Console.WriteLine($"  Value: {distroNameValue}");
        Console.WriteLine($"  Upstream default: {UpstreamDistroValue}");
        Console.WriteLine($"  Dash0 expected:   {ExpectedDash0DistroValue}");

        Console.WriteLine();

        bool caseAPassed = markerValue == ExpectedCaseAMarker;
        bool caseBPassed = distroNameValue == ExpectedDash0DistroValue;

        Console.WriteLine($"Case A verdict: {(caseAPassed ? "PASS" : "FAIL")}");
        Console.WriteLine($"Case B verdict: {(caseBPassed ? "PASS" : "FAIL")}");

        return (caseAPassed && caseBPassed) ? 0 : 1;
    }
}
