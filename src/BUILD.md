---
name: Build & test guide
description: How to build, test, and lint prompto locally. Use the same commands CI runs.
---

# Build & test

All commands run from `src/`. Go version is pinned in `go.mod`.

## Build

```bash
go build -o /path/to/prompto
```

Drop the binary anywhere on `$PATH`. For local iteration:

```bash
go build -o ~/bin/prompto
```

`prompto --version` on an unreleased build reports `0.0.0-dev`.

## Test

```bash
go test -count=1 ./...                       # unit tests
go test -race -count=3 ./daemon/... ./log/... ./prompt/... # race tests (required on daemon/, log/, prompt/)
```

The `-race` run is a hard gate in CI for `daemon/`, `log/` and `prompt/`, which
all have goroutines over shared state. Don't merge code that fails race tests.

## Lint

All lint tools are pinned via the `tool` directive in `go.mod` and run via
`go tool ...`. CI runs the same binaries. Bump a tool with:

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@vX.Y.Z
```

To run everything CI runs locally:

```bash
go tool golangci-lint run
go list ./... | grep -v '/daemon/ipc$' | xargs go tool fieldalignment
go list ./... | grep -v '/daemon/ipc$' | xargs go tool modernize
```

`/daemon/ipc` is excluded because it contains generated protobuf code that
upstream tools cannot satisfy.

## Markdown lint

```bash
# from the repo root, not src/
cd .. && npx -y markdownlint-cli2@0.20.0 '**/*.md'
```

The cli2 version is pinned in two places (bump together):

- CI: `.github/workflows/markdown.yml` (action SHA bundles cli2 v0.20.0).
- Local: the npx command above.

Rules live in `.markdownlint-cli2.yaml` at the repo root.

## Generated code

The protobuf wire format lives in `daemon/daemon.proto`. After editing it:

```bash
go generate ./...
```

CI fails if generated files are stale. Run `go generate` and commit the
regenerated `daemon/ipc/*.pb.go` files.

## Full pre-push check

```bash
go generate ./...           # regenerate if .proto changed
go tool golangci-lint run
go list ./... | grep -v '/daemon/ipc$' | xargs go tool fieldalignment
go list ./... | grep -v '/daemon/ipc$' | xargs go tool modernize
go test -count=1 ./...
go test -race -count=3 ./daemon/... ./log/... ./prompt/...
(cd .. && npx -y markdownlint-cli2@0.20.0 '**/*.md')
```

If those pass locally, CI passes. They are byte-identical to what runs in
`.github/workflows/{build_code,code,markdown}.yml`.
