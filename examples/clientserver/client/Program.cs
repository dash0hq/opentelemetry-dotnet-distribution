var serverUrl = Environment.GetEnvironmentVariable("SERVER_URL") ?? "http://server:8080/";

using var httpClient = new HttpClient();
var random = new Random();
var tasks = new List<Task>();

for (int i = 0; i < 6; i++)
{
    int threadId = i;
    tasks.Add(Task.Run(async () =>
    {
        var localRandom = new Random(Guid.NewGuid().GetHashCode());
        while (true)
        {
            try
            {
                var response = await httpClient.GetAsync(serverUrl);
                var body = await response.Content.ReadAsStringAsync();
                Console.WriteLine($"[thread {threadId}] {response.StatusCode}: {body}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"[thread {threadId}] request failed: {ex.Message}");
            }

            var sleepMs = localRandom.Next(500, 3001);
            await Task.Delay(sleepMs);
        }
    }));
}

await Task.WhenAll(tasks);
