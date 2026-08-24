using Microsoft.EntityFrameworkCore;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();
var logger = app.Logger;

var connectionString = Environment.GetEnvironmentVariable("POSTGRES_CONNECTION_STRING")
    ?? "Host=postgres;Port=5432;Username=postgres;Password=postgres;Database=postgres";

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// Exercises the EntityFrameworkCore instrumentation (via the Npgsql EF Core provider): a DbContext operation
// creates the schema on first use, inserts a row, and counts the total.
app.MapGet("/query", async () =>
{
    var options = new DbContextOptionsBuilder<HitsContext>()
        .UseNpgsql(connectionString)
        .Options;

    await using var db = new HitsContext(options);
    await db.Database.EnsureCreatedAsync();

    db.Hits.Add(new Hit { CreatedAt = DateTimeOffset.UtcNow });
    await db.SaveChangesAsync();

    var count = await db.Hits.CountAsync();
    logger.LogInformation("Total hits recorded: {Count}", count);
    return Results.Ok(new { totalHits = count });
});

app.Run();

class HitsContext : DbContext
{
    public HitsContext(DbContextOptions<HitsContext> options) : base(options) { }

    public DbSet<Hit> Hits => Set<Hit>();
}

class Hit
{
    public int Id { get; set; }
    public DateTimeOffset CreatedAt { get; set; }
}
