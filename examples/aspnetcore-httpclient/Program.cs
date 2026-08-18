var builder = WebApplication.CreateBuilder(args);

builder.Services.AddHttpClient("self", client =>
{
    var selfUrl = Environment.GetEnvironmentVariable("SELF_URL") ?? "http://localhost:8080";
    client.BaseAddress = new Uri(selfUrl);
});

var app = builder.Build();
var logger = app.Logger;

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// Simulates a downstream service. Called by /call via HttpClient, exercising both the inbound ASP.NET Core
// instrumentation (for this request) and the outbound HttpClient instrumentation (from the caller's perspective).
app.MapGet("/downstream", () =>
{
    logger.LogInformation("Handling downstream request");
    return Results.Ok(new { message = "hello from downstream", timestamp = DateTimeOffset.UtcNow });
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
