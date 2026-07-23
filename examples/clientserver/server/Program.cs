var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var random = new Random();

app.MapGet("/", async () =>
{
    var delayMs = random.Next(100, 901); // 0.1–0.9s
    await Task.Delay(delayMs);
    var value = random.Next(0, 1_000_000);
    return Results.Ok(new { value, delayMs });
});

app.Run("http://0.0.0.0:8080");
