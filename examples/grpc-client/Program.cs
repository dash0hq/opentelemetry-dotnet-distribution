using Grpc.Net.Client;
using GrpcClient;
using Microsoft.AspNetCore.Server.Kestrel.Core;

// gRPC needs HTTP/2 client-side even without TLS; Kestrel's default protocol
// selection for a non-TLS endpoint is HTTP/1.1 only (ALPN normally picks the
// protocol, which needs TLS), so both sides need explicit opt-in below.
AppContext.SetSwitch("System.Net.Http.SocketsHttpHandler.Http2UnencryptedSupport", true);

var builder = WebApplication.CreateBuilder(args);

// A single combined Http1AndHttp2 endpoint without TLS can't reliably
// negotiate h2c prior-knowledge for a plain HttpClient-based gRPC call
// (Kestrel sent back "HTTP/2 error code PROTOCOL_ERROR"), so the gRPC
// service gets its own dedicated HTTP/2-only endpoint on a separate,
// internal-only port; REST endpoints (readiness probe, /call) stay on 8080.
builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(8080, o => o.Protocols = HttpProtocols.Http1);
    options.ListenAnyIP(8081, o => o.Protocols = HttpProtocols.Http2);
});

builder.Services.AddGrpc();

var app = builder.Build();

app.MapGrpcService<GreeterService>();

app.MapGet("/", () => Results.Ok(new { status = "ok" }));

// Exercises the Grpc.Net.Client instrumentation: calls the gRPC service hosted in this same process.
app.MapGet("/call", async () =>
{
    var channel = GrpcChannel.ForAddress("http://localhost:8081");
    var client = new Greeter.GreeterClient(channel);
    var reply = await client.SayHelloAsync(new HelloRequest { Name = "world" });
    return Results.Ok(new { message = reply.Message });
});

app.Run();
