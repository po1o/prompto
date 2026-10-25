---
name: Build & test guide
description: How to build, test, and lint prompto locally. The same commands CI runs.
---

# Build & test

Everything is driven by [go-task](https://taskfile.dev). Tool versions are
pinned via the `tool` directive in `src/go.mod` and invoked through `go tool`,
so local and CI runs are byte-identical. Go's version is pinned in `go.mod`.

Invoke tasks with a global `task` binary, or through Go (no install needed):

```bash
go tool task <name>     # or: task <name>
```

## Common tasks

```bash
go tool task build       # build ./bin/prompto (override with OUTPUT=/path/to/prompto)
go tool task test:unit   # go test -count=1 ./...
go tool task test:race   # go test -race -count=3 ./daemon/... ./log/... ./prompt/...
go tool task test        # the full CI check suite (see below)
```

`prompto --version` on an unreleased build reports `0.0.0-dev`.

## Full pre-push check

```bash
go tool task test
```

`test` runs, in order:

- `go vet ./...`, then `GOOS=freebsd GOARCH=amd64 go vet ./...` (typechecks the freebsd-tagged tests)
- `go tool golangci-lint run`
- `go list ./... | grep -v '/daemon/ipc$' | xargs go tool modernize`
- `go list ./... | grep -v '/daemon/ipc$' | xargs go tool fieldalignment`
- `go test -count=1 ./...`
- `go test -race -count=3 ./daemon/... ./log/... ./prompt/...`

`/daemon/ipc` is excluded from `modernize` and `fieldalignment` because it
contains generated protobuf code that upstream tools cannot satisfy.

If `go tool task test` passes locally, CI passes — `.github/workflows/ci.yml`
runs the same `go tool task test` on Linux and macOS.

## Generated code

The protobuf wire format lives in `daemon/daemon.proto`. After editing it (or
bundled themes/shell scripts):

```bash
go tool task generate
```

CI runs `go tool task generate:check` and fails if generated files are stale.
Run `go tool task generate` and commit the regenerated files.

## Bumping a pinned tool

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@vX.Y.Z
go mod tidy
```

## Markdown lint

From the repo root (not `src/`):

```bash
task markdown-lint       # npx -y markdownlint-cli2@<MDLINT_VERSION> '**/*.md'
```

Rules live in `.markdownlint-cli2.yaml` at the repo root. The cli2 version is
pinned in `Taskfile.yml` (`MDLINT_VERSION`) and `.github/workflows/markdown.yml`
— bump them together.
