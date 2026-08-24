# SPDX-FileCopyrightText: Copyright 2026 Dash0 Inc.
# SPDX-License-Identifier: Apache-2.0

# The jammy variant (Ubuntu 22.04) is needed at build time only: Grpc.Tools'
# bundled arm64 protoc binary requires a newer glibc than the default
# bullseye-based SDK image ships (GLIBC_2.32+ vs bullseye's 2.31).
FROM mcr.microsoft.com/dotnet/sdk:6.0-jammy AS build
WORKDIR /src
COPY GrpcClient.csproj .
RUN dotnet restore
COPY . .
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:6.0
WORKDIR /app
COPY --from=build /app .
EXPOSE 8080
ENTRYPOINT ["dotnet", "GrpcClient.dll"]
