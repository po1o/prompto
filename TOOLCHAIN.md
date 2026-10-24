# Toolchain

Every tool runs through [go-task](https://taskfile.dev), pinned in `src/go.mod` via the `tool` directive, so local
and CI use byte-identical binaries. The rule: **`task test` green locally ⇒ CI green.** Deep rationale lives in the
code/workflow comments — this file is just the map.

## Layout

- `Taskfile.yml` — every task (repo root).
- `src/` — all Go code; `go.mod` holds the module, Go version, and pinned `tool` directives.
  Module path `github.com/po1o/prompto/src`.
- `.github/workflows/` — CI (table below).
- Lint config: `.commitlintrc.cjs`, `.markdownlint-cli2.yaml`, `src/.golangci.yml`.

## Tasks

Tools are pinned in `src/go.mod`, run via `go tool <x>` — nothing installed globally (bump with
`go get -tool <pkg>@<ver> && go mod tidy`). `task` lists them all; the essentials:

| Task | Does |
| --- | --- |
| `test` | the CI gate: vet → lint → modernize → fieldalignment → unit tests → race tests |
| `test:unit` / `test:race` | unit tests / race detector on `daemon`, `log`, `prompt` |
| `build` / `build:all` | host binary / all 5 cross-compiled targets |
| `generate` / `generate:check` | `go generate ./...` / regenerate + fail on diff |
| `tidy:check` | `go mod tidy` + fail on diff |
| `markdown-lint` / `commit-lint` | the two Node (`npx`) checks |
| `setup` (+ `setup:*`) | one-time repo bootstrap — see below |

Want a bare `task`? `brew install go-task/tap/go-task` and drop the `go tool` prefix.

## CI

Each workflow installs pinned Go via `bootstrap-go`, then runs a task — no logic duplicated.

| Workflow (display) | Runs | Required check (context) |
| --- | --- | --- |
| `ci.yml` (Code) | `detect-changes` → `test` (Linux/macOS) ∥ `build` → aggregator | **`post-check`** |
| `gomod.yml` (Dependencies) | `task tidy:check` | `tidy-check` |
| `commits.yml` (Commit-PR) | commitlint — commit messages + PR title | `convention` |
| `markdown.yml` (Markdown) | markdownlint-cli2 | `check` |
| `dependabot_tidy.yml` / `dependabot_automerge.yml` | Dependabot fixups | — |

**One gate, no deadlock.** In `ci.yml`, `test`/`build` are gated by the `detect-changes` detector (job-level `if`),
not a workflow-level `paths-ignore`. So a docs-only PR *skips* them and the single required **`post-check`**
aggregator still reports success — whereas a `paths-ignore`'d required workflow would hang "Pending" forever.

## Setup (once per new repo, needs `gh` admin)

`task setup` runs `setup:branch-protection` + `setup:merge-settings`:

- **branch-protection** — a `protected` ruleset: PR required, **1 approving review**, the 4 checks, squash/rebase
  only, no force-push/delete. Bypassed by **admins** and the **auto-merge App** — so only Dependabot merges
  review-free. Needs the App's numeric ID in the repo variable `TIDY_APP_ID`.
- **merge-settings** — auto-merge on, merge commits off, delete head branch on merge, squash commit = PR title +
  commit details.

Separate (1Password, not in the umbrella so `setup` stays op-free):

- **`setup:app-credentials`** — streams the App's `TIDY_APP_CLIENT_ID` / `TIDY_APP_PRIVATE_KEY` (Dependabot secrets) and
  `TIDY_APP_ID` (variable) from `op` into `gh` over a pipe — never on disk, in args, or in the environment. Export
  `OP_CLIENT_ID_REF` / `OP_PRIVATE_KEY_REF` / `OP_APP_ID_REF` first.

Still manual: create the GitHub App once, and install it on each repo.

## Dependabot

Daily grouped PRs (`.github/dependabot.yml`); enable it in **Settings → Code security**. Chain: Dependabot opens a
PR → `dependabot_tidy.yml` runs `go mod tidy` and pushes (App-token push re-triggers CI) → `dependabot_automerge.yml`
arms auto-merge → checks go green → merges. **Patch/minor auto-merge; majors wait for review.** It uses an App
token (not `GITHUB_TOKEN`) because the App key is a Dependabot secret unreadable by human PRs — making the App the
*only* thing that can skip the review. Full reasoning is in each workflow's header.

## Creating the GitHub App

The Dependabot workflows push commits to, and merge, PRs — things the default `GITHUB_TOKEN` can't do on Dependabot
PRs. They mint a short-lived token from a **GitHub App you own**. The App named in this repo (`po1o-tidy-bot`) is
mine; **a third party must create their own** — you can't use mine. Once, reusable across all your repos:

1. **New App** — Settings → Developer settings → GitHub Apps → *New*. Repository permissions:
   **Contents: Read and write** + **Pull requests: Read and write** (nothing else); uncheck Webhook → Active.
   Note the **App ID** (a number) and **Client ID** (`Iv23li…`), then generate a **private key** (`.pem`).
2. **Install** it on each repo (the App's page → Install App).
3. **Provide the credentials** per repo — via `task setup:app-credentials` (from 1Password) or by hand: two
   **Dependabot** secrets `TIDY_APP_CLIENT_ID` / `TIDY_APP_PRIVATE_KEY`, and one repo **Actions variable**
   `TIDY_APP_ID` (the numeric App ID, used by the ruleset bypass).

The two are **Dependabot** secrets (not Actions) on purpose: only Dependabot-triggered runs can read them, so a
human PR can never mint the token — which is exactly what keeps the review-bypass Dependabot-only.

## Typical change

```bash
task test                      # full gate, before pushing
git commit -m "feat(x): …"     # Conventional Commit
# open a PR — CI runs the same task test
```
