using Quartz;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddQuartz(q =>
{
    var jobKey = new JobKey("example-job");
    q.AddJob<ExampleJob>(opts => opts.WithIdentity(jobKey));
    q.AddTrigger(opts => opts
        .ForJob(jobKey)
        .WithIdentity("example-trigger")
        .StartNow()
        .WithSimpleSchedule(x => x.WithIntervalInSeconds(2).RepeatForever()));
});
builder.Services.AddQuartzHostedService(opts => opts.WaitForJobsToComplete = true);

var app = builder.Build();

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

app.Run();

// Exercises the Quartz instrumentation: fires every 2 seconds, starting immediately on app startup.
class ExampleJob : IJob
{
    private readonly ILogger<ExampleJob> _logger;

    public ExampleJob(ILogger<ExampleJob> logger)
    {
        _logger = logger;
    }

    public Task Execute(IJobExecutionContext context)
    {
        _logger.LogInformation("Example job executed");
        return Task.CompletedTask;
    }
}
