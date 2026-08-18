using Npgsql;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();
var logger = app.Logger;

var connectionString = Environment.GetEnvironmentVariable("POSTGRES_CONNECTION_STRING")
    ?? "Host=postgres;Port=5432;Username=postgres;Password=postgres;Database=postgres";

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// Opens a connection, creates a table on first use, inserts a row and counts the total. Exercises the Npgsql ADO.NET
// instrumentation (connection open, command execution spans).
app.MapGet("/query", async () =>
{
    await using var connection = new NpgsqlConnection(connectionString);
    await connection.OpenAsync();

    await using (var createCmd = new NpgsqlCommand(
        "CREATE TABLE IF NOT EXISTS dash0_example_hits (id SERIAL PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now())",
        connection))
    {
        await createCmd.ExecuteNonQueryAsync();
    }

    await using (var insertCmd = new NpgsqlCommand("INSERT INTO dash0_example_hits DEFAULT VALUES", connection))
    {
        await insertCmd.ExecuteNonQueryAsync();
    }

    long count;
    await using (var countCmd = new NpgsqlCommand("SELECT COUNT(*) FROM dash0_example_hits", connection))
    {
        count = (long)(await countCmd.ExecuteScalarAsync())!;
    }

    logger.LogInformation("Total hits recorded: {Count}", count);
    return Results.Ok(new { totalHits = count });
});

app.Run();
