# Agent Instructions

## General File Creation Guidelines

When creating new files:

- **Always use LF (Unix-style) line endings**, not CRLF (Windows-style)
- This repository uses `.gitattributes` to enforce LF line endings
- Ensures consistency across all platforms and avoids Git warnings

## Golang

When editing Go files (`*.go`):

- Read `.github/instructions/golang.md` and announce once per task that you are following it.
- Before committing, run the lint and test suite from `src/`:
  - `go tool golangci-lint run`
  - `` go list ./... | grep -v '/daemon/ipc$' | xargs go tool fieldalignment ``
  - `` go list ./... | grep -v '/daemon/ipc$' | xargs go tool modernize ``
  - `go test -count=1 ./...`
  - `go test -race -count=3 ./daemon/... ./log/...`
- Lint-tool versions are pinned in `src/go.mod` via the `tool` directive — CI
  runs the same binaries. **Never** install a different version with
  `go install ...@latest`; bump via `go get -tool <pkg>@<ver>` so CI follows.
- Full build/test reference: [`src/BUILD.md`](src/BUILD.md).

## Markdown

When editing Markdown (`*.md`, `*.mdx`):

- Read `.github/instructions/markdown.md` and announce once per task that you are following it.
- Use proper headings (`##`, `###`), fenced code blocks with language, and keep lines within the configured limit.
- Lint locally from the **repo root** (not `src/`):
  `npx -y markdownlint-cli2@0.20.0 '**/*.md'`
- Config: `.markdownlint-cli2.yaml` (line length 120; MD013 disabled for tables; MD060 disabled).

## PowerShell

When editing PowerShell files (`*.ps1`, `*.psm1`, `*.psd1`):

- Read `.github/instructions/powershell.md` and announce once per task that you are following it.
- Follow PowerShell best practices for naming, formatting, and error handling.
- Include comment-based help for public functions and ensure proper parameter validation.

## Commit and Pull Requests Guidelines

- Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) for PR titles and commit messages.
- The repository-specific rules are in `.commitlintrc.yml` (not `.json`).
- **Allowed commit types** (anything else fails CI):
  `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`, `theme`.
  Note: `build` is **not** allowed — use `ci(deps):` or `chore(deps):` for tooling/dependency bumps.
- Always run the full lint + test suite (see Golang section) before pushing.
- Limit body lines to 200 characters; the rule is enforced by commitlint.
- **Do not commit initial plans or progress updates as separate commits.**
  Include planning information in the PR description instead.

Examples:

- `feat(config): cache remote configs via HEAD check`
- `fix(markdown): correct reference link syntax in docs`
- `ci(deps): pin golangci-lint via go.mod tool directive`
