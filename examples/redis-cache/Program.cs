using StackExchange.Redis;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();
var logger = app.Logger;

var connectionString = Environment.GetEnvironmentVariable("REDIS_CONNECTION_STRING") ?? "redis:6379";

// A single, lazily-connected multiplexer shared across requests, per StackExchange.Redis's own guidance.
var redis = new Lazy<ConnectionMultiplexer>(() => ConnectionMultiplexer.Connect(connectionString));

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// Increments a counter in Redis on every call. Exercises the StackExchange.Redis instrumentation (command spans).
app.MapGet("/cache", async () =>
{
    var db = redis.Value.GetDatabase();
    var newValue = await db.StringIncrementAsync("dash0:example:counter");
    logger.LogInformation("Incremented Redis counter to {Value}", newValue);
    return Results.Ok(new { counter = newValue });
});

app.Run();
