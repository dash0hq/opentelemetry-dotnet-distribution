var builder = WebApplication.CreateBuilder(args);

builder.Services.AddHttpClient("self", client =>
{
    var selfUrl = Environment.GetEnvironmentVariable("SELF_URL") ?? "http://localhost:8080";
    client.BaseAddress = new Uri(selfUrl);
});

var app = builder.Build();
var logger = app.Logger;

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// net8.0 twin of aspnetcore-httpclient, used to validate that instrumentation-image packaging changes don't
// regress ASP.NET Core tracing for TFMs above the net6.0/net7.0 floor (see Directory.Packages.props in
// opentelemetry-dotnet-instrumentation dash0-main, which pins OpenTelemetry.Instrumentation.AspNetCore
// differently per TFM).
app.MapGet("/downstream", () =>
{
    logger.LogInformation("Handling downstream request");
    return Results.Ok(new { message = "hello from downstream (net8.0)", timestamp = DateTimeOffset.UtcNow });
});

app.MapGet("/call", async (IHttpClientFactory httpClientFactory) =>
{
    logger.LogInformation("Calling downstream endpoint via HttpClient");
    var client = httpClientFactory.CreateClient("self");
    var response = await client.GetAsync("/downstream");
    response.EnsureSuccessStatusCode();
    var body = await response.Content.ReadAsStringAsync();
    return Results.Content(body, "application/json");
});

app.Run();
