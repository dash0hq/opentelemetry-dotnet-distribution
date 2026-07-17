# Contributing

## PoC-phase workflow

This repo is currently in PoC. Development happens directly on `main` — no PR gate.
Once v1 is closer, we'll introduce a PR-based workflow with CI gates.

## Getting set up

```sh
git clone --recurse-submodules https://github.com/dash0hq/opentelemetry-dotnet-distro.git
cd opentelemetry-dotnet-distro
dotnet restore
dotnet build
```

Prerequisites: .NET SDK 9.0.x (SDK 10 preferred once available).

## Running a build

```sh
dotnet run --project build/Build -- --rid linux-x64 --version 0.0.1-dev
```

Bundle lands at `artifacts/dash0-opentelemetry-dotnet-autoinstrumentation-<rid>.tar.gz`.

## Working with the runtime-support branch

The runtime-support-track (`.NET 6+` patches on the upstream instrumentation code)
lives on the [`dash0-main`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation/tree/dash0-main)
branch of `dash0hq/opentelemetry-dotnet-instrumentation`, consumed here via a
git submodule under `forks/` (added in U4 of the plan).

To rebase the runtime-support branch onto a new upstream release:
1. Cut a working branch on the fork off the new upstream tag.
2. `git rebase <new-tag>` — resolve conflicts using standard git tooling.
3. Update this repo's submodule pin and `Directory.Packages.props` in lockstep.

Full rebase procedure will live in `docs/rebase-runbook.md` (U4 in the plan).

## Repo conventions

- **Commits** use conventional-commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `build:`, `ci:`, `chore:`.
- **File paths** in docs and plans are always repo-relative — never absolute.
- **Plan and design documents** live under `docs/`. Read [the plan](docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md) before starting a unit of work — it defines scope, requirements traceability, and test scenarios per unit.
- **Runtime-support-track patches** on the fork use the reason code `dash0-carry` in `.dash0-branch-meta.yaml`.

## Where to look

- Design: [docs/brainstorms/](docs/brainstorms/)
- Implementation plan and per-unit specs: [docs/plans/](docs/plans/)
- Spike outcomes: [docs/spikes/](docs/spikes/), [spike/](spike/)
