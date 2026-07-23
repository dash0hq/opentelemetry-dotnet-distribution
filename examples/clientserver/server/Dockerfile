FROM ubuntu:22.04 AS build
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

RUN apt-get update && apt-get install -y dotnet-sdk-6.0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY server.csproj .
RUN dotnet restore
COPY . .
RUN dotnet publish -c Release -o /app

FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates
RUN apt-get update && apt-get install -y aspnetcore-runtime-6.0 tar \
    && rm -rf /var/lib/apt/lists/*

ENV HOME=/root

WORKDIR /app
COPY --from=build /app .
RUN chmod +x /app/server

EXPOSE 8080
CMD mkdir -p $HOME/.otel-dotnet-auto && \
    tar xzvf /shared/opentelemetry-dotnet-instrumentation-linux.tar.gz -C $HOME/.otel-dotnet-auto && \
    chmod +x $HOME/.otel-dotnet-auto/instrument.sh && \
    . $HOME/.otel-dotnet-auto/instrument.sh && \
    ./server
