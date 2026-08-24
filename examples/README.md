# Examples

Small .NET 6 apps for exercising the instrumentations shipped by this distribution end to end, against a real
Kubernetes cluster instrumented by the [Dash0 operator](https://github.com/dash0hq/dash0-operator).

Each app is a plain, uninstrumented ASP.NET Core minimal API — no Dash0/OpenTelemetry code or packages. Instrumentation
is applied at the pod-admission level by the Dash0 operator's mutating webhook once the target namespace carries a
`Dash0Monitoring` resource, exactly like the operator's own Node.js test app under `dash0-operator/test-resources/`.
This means these apps double as a way to validate the operator + this distribution working together, not just the
distribution in isolation.

| App | Directory | Instrumentation exercised |
| --- | --- | --- |
| ASP.NET Core + HttpClient | [`aspnetcore-httpclient`](aspnetcore-httpclient) | Inbound ASP.NET Core requests, outbound `HttpClient` calls, `ILogger` log correlation |
| ADO.NET (Npgsql/PostgreSQL) | [`sqlclient-postgres`](sqlclient-postgres) | Npgsql command execution spans |
| StackExchange.Redis | [`redis-cache`](redis-cache) | Redis command spans |

All three target `net6.0` specifically, to validate the net6.0 runtime support carried on the `dash0-main` fork (see
the root [README](../README.md)).

## Building and pushing images

These instructions assume the [local kind cluster + registry setup](https://github.com/dash0hq/dash0-operator/blob/main/CONTRIBUTING.md#setting-up-a-kind-cluster-for-local-testing)
described in `dash0-operator`'s `CONTRIBUTING.md`, with the registry reachable at `localhost:5001`. Adjust the image
references in each `k8s/deployment.yaml` if you're using a different registry.

```sh
for app in aspnetcore-httpclient sqlclient-postgres redis-cache; do
  docker build -t "localhost:5001/dash0-dotnet-example-${app}:latest" "examples/${app}"
  docker push "localhost:5001/dash0-dotnet-example-${app}:latest"
done
```

## Deploying

```sh
kubectl apply -f examples/namespace.yaml
kubectl apply -f examples/aspnetcore-httpclient/k8s/deployment.yaml
kubectl apply -f examples/sqlclient-postgres/k8s/postgres.yaml
kubectl apply -f examples/sqlclient-postgres/k8s/deployment.yaml
kubectl apply -f examples/redis-cache/k8s/redis.yaml
kubectl apply -f examples/redis-cache/k8s/deployment.yaml
```

`examples/namespace.yaml` creates the `dash0-dotnet-examples` namespace and a `Dash0Monitoring` resource targeting it
with `instrumentWorkloads.mode: all`. The Dash0 operator must already be installed in the cluster (see the root
README of `dash0-operator`) for that resource to have any effect.

## Exercising the apps

```sh
kubectl -n dash0-dotnet-examples port-forward svc/aspnetcore-httpclient 8080:8080 &
curl http://localhost:8080/call

kubectl -n dash0-dotnet-examples port-forward svc/sqlclient-postgres 8081:8080 &
curl http://localhost:8081/query

kubectl -n dash0-dotnet-examples port-forward svc/redis-cache 8082:8080 &
curl http://localhost:8082/cache
```

Each call should produce a trace in Dash0 (or whichever OTLP backend the operator/collector is configured to export
to) spanning the inbound HTTP request plus the downstream call it makes (HttpClient, Npgsql, or Redis, respectively).
