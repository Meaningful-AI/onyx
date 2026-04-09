# Onyx Developer Script

[![Deploy Status](https://github.com/onyx-dot-app/onyx/actions/workflows/release-devtools.yml/badge.svg)](https://github.com/onyx-dot-app/onyx/actions/workflows/release-devtools.yml)
[![PyPI](https://img.shields.io/pypi/v/onyx-devtools.svg)](https://pypi.org/project/onyx-devtools/)

`ods` is [onyx.app](https://github.com/onyx-dot-app/onyx)'s devtools utility script.
It is packaged as a python [wheel](https://packaging.python.org/en/latest/discussions/package-formats/) and available from [PyPI](https://pypi.org/project/onyx-devtools/).

## Installation

A stable version of `ods` is provided in the default [python venv](https://github.com/onyx-dot-app/onyx/blob/main/CONTRIBUTING.md#backend-python-requirements)
which is synced automatically if you have [pre-commit](https://github.com/onyx-dot-app/onyx/blob/main/CONTRIBUTING.md#formatting-and-linting)
hooks installed.

While inside the Onyx repository, activate the root project's venv,

```shell
source .venv/bin/activate
```

### Prerequisites

Some commands require external tools to be installed and configured:

- **Docker** - Required for `compose`, `logs`, and `pull` commands
  - Install from [docker.com](https://docs.docker.com/get-docker/)

- **uv** - Required for `backend` commands
  - Install from [docs.astral.sh/uv](https://docs.astral.sh/uv/)

- **GitHub CLI** (`gh`) - Required for `run-ci`, `cherry-pick`, `trace`, `pr-review`, and `pr-checks` commands
  - Install from [cli.github.com](https://cli.github.com/)
  - Authenticate with `gh auth login`

- **AWS CLI** - Required for `screenshot-diff` commands and `journey publish` (S3 artifact sync)
  - Install from [aws.amazon.com/cli](https://aws.amazon.com/cli/)
  - Authenticate with `aws sso login` or `aws configure`

### Autocomplete

`ods` provides autocomplete for `bash`, `fish`, `powershell` and `zsh` shells.

For more information, see `ods completion <shell> --help` for your respective `<shell>`.

#### zsh

_Linux_

```shell
ods completion zsh | sudo tee "${fpath[1]}/_ods" > /dev/null
```

_macOS_

```shell
ods completion zsh > $(brew --prefix)/share/zsh/site-functions/_ods
```

#### bash

```shell
ods completion bash | sudo tee /etc/bash_completion.d/ods > /dev/null
```

_Note: bash completion requires the [bash-completion](https://github.com/scop/bash-completion/) package be installed._

## Commands

### `compose` - Launch Docker Containers

Launch Onyx docker containers using docker compose.

```shell
ods compose [profile]
```

**Profiles:**

- `dev` - Use dev configuration (exposes service ports for development)
- `multitenant` - Use multitenant configuration

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--down` | `false` | Stop running containers instead of starting them |
| `--wait` | `true` | Wait for services to be healthy before returning |
| `--force-recreate` | `false` | Force recreate containers even if unchanged |
| `--tag` | | Set the `IMAGE_TAG` for docker compose (e.g. `edge`, `v2.10.4`) |

**Examples:**

```shell
# Start containers with default configuration
ods compose

# Start containers with dev configuration
ods compose dev

# Start containers with multitenant configuration
ods compose multitenant

# Stop running containers
ods compose --down
ods compose dev --down

# Start without waiting for services to be healthy
ods compose --wait=false

# Force recreate containers
ods compose --force-recreate

# Use a specific image tag
ods compose --tag edge
```

### `logs` - View Docker Container Logs

View logs from running Onyx docker containers. Service names are available as
arguments to filter output, with tab-completion support.

```shell
ods logs [service...]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--follow` | `true` | Follow log output |
| `--tail` | | Number of lines to show from the end of the logs |

**Examples:**

```shell
# View logs from all services (follow mode)
ods logs

# View logs for a specific service
ods logs api_server

# View logs for multiple services
ods logs api_server background

# View last 100 lines and follow
ods logs --tail 100 api_server

# View logs without following
ods logs --follow=false
```

### `pull` - Pull Docker Images

Pull the latest images for Onyx docker containers.

```shell
ods pull
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--tag` | | Set the `IMAGE_TAG` for docker compose (e.g. `edge`, `v2.10.4`) |

**Examples:**

```shell
# Pull images
ods pull

# Pull images with a specific tag
ods pull --tag edge
```

### `backend` - Run Backend Services

Run backend services (API server, model server) with environment loaded from
`.vscode/.env`. On first run, copies `.vscode/env_template.txt` to `.vscode/.env`
if the `.env` file does not already exist.

Enterprise Edition features are enabled by default with license enforcement
disabled, matching the `compose` command behavior.

```shell
ods backend <subcommand>
```

**Subcommands:**

- `api` - Start the FastAPI backend server (`uvicorn onyx.main:app --reload`)
- `model_server` - Start the model server (`uvicorn model_server.main:app --reload`)

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--no-ee` | `false` | Disable Enterprise Edition features (enabled by default) |
| `--worktree` | current checkout | Run the command against a tracked agent-lab worktree |
| `--port` | `8080` (api) / `9000` (model_server) | Port to listen on |

Shell environment takes precedence over `.env` file values, so inline overrides
work as expected (e.g. `S3_ENDPOINT_URL=foo ods backend api`).

When run inside a tracked `agent-lab` worktree, `ods backend api` and
`ods backend model_server` will automatically use that worktree's reserved
ports unless you override them explicitly with `--port`.

The same command can also be launched from the `codex/agent-lab` control
checkout against another tracked worktree via `--worktree <branch>`.

**Examples:**

```shell
# Start the API server
ods backend api

# Start the API server on a custom port
ods backend api --port 9090

# Start without Enterprise Edition
ods backend api --no-ee

# Start the model server
ods backend model_server

# Start the model server on a custom port
ods backend model_server --port 9001

# Run the API server for a tracked product worktree from the control checkout
ods backend api --worktree codex/fix/auth-banner-modal
```

### `web` - Run Frontend Scripts

Run npm scripts from `web/package.json` without manually changing directories.

```shell
ods web <script> [args...]
```

Script names are available via shell completion (for supported shells via
`ods completion`), and are read from `web/package.json`.

When run inside a tracked `agent-lab` worktree, `ods web ...` automatically
injects the worktree's `PORT`, `BASE_URL`, `WEB_DOMAIN`, `INTERNAL_URL`, and
`MCP_INTERNAL_URL` so the Next.js dev server boots against the right isolated
stack.

From the `codex/agent-lab` control checkout, `--worktree <branch>` applies the
same wiring to a tracked target worktree.

**Examples:**

```shell
# Start the Next.js dev server
ods web dev

# Run web lint task
ods web lint

# Forward extra args to the script
ods web test --watch

# Run the Next.js dev server for a tracked product worktree
ods web dev --worktree codex/fix/auth-banner-modal
```

### `worktree` - Manage Agent-Lab Worktrees

Create and manage local git worktrees for agentized development. Each tracked
worktree gets:

- a reserved port bundle for web, API, model server, and MCP
- an explicit dependency mode for local external state
- generated `.vscode/.env.agent-lab` and `.vscode/.env.web.agent-lab` files
- a local artifact directory for verification logs and summaries
- a manifest stored under the shared git metadata directory
- bootstrap support for env files, Python runtime, and frontend dependencies

`ods worktree create` is the authoritative entrypoint for this workflow. Do not
use raw `git worktree add` when you want the `agent-lab` harness, because you
will skip the manifest, env overlays, dependency bootstrap, and lane-aware base
selection.

```shell
ods worktree <subcommand>
```

**Subcommands:**

- `create <branch>` - Create a worktree and manifest
- `bootstrap [worktree]` - Prepare env files and dependencies for a worktree
- `deps up|status|reset|down [worktree]` - Provision and manage namespaced external state
- `status` - List tracked worktrees and URLs
- `show [worktree]` - Show detailed metadata for one worktree
- `remove <worktree>` - Remove a worktree and its local state

`ods worktree create` bootstraps new worktrees by default. The current bootstrap
behavior is:

- link `.vscode/.env` and `.vscode/.env.web` from the source checkout when present
- link the source checkout's `.venv` when present
- clone `web/node_modules` into the worktree when present, falling back to
  `npm ci --prefer-offline --no-audit`

Current isolation boundary:

- worktree-local: web/API/model-server ports, URLs, env overlays, artifact dirs
- namespaced when `--dependency-mode namespaced` is used: PostgreSQL database,
  Redis prefix, and MinIO file-store bucket
- always shared: OpenSearch/Vespa and the rest of the docker-compose dependency stack

`namespaced` is the default dependency mode on `agent-lab`. `shared` is still
available for lighter-weight work that does not need isolated DB/Redis/MinIO
state.

Branch lanes:

- `codex/lab/<name>` worktrees are treated as harness work and default to
  `codex/agent-lab` as the base ref
- `codex/fix/<name>`, `codex/feat/<name>`, and other conventional product lanes
  default to `origin/main` as the base ref
- branches that do not encode a lane fall back to `HEAD`; use `--from` or a
  clearer branch name when the base matters

Control-plane note:

- the harness lives on `codex/agent-lab`
- product worktrees can still be based on `origin/main`
- run `ods backend`, `ods web`, `ods verify`, and `ods agent-check` with
  `--worktree <branch>` from the control checkout when the target worktree does
  not carry the harness code itself

Search/vector note:

- OpenSearch/Vespa stay shared-only
- this branch intentionally does not implement namespaced or per-worktree search stacks
- tasks that touch search/index infrastructure should assume a shared surface

**Examples:**

```shell
# Create a product bugfix worktree from main
ods worktree create codex/fix/auth-banner-modal

# Create a lab-only worktree from agent-lab
ods worktree create codex/lab/browser-validation

# Reuse the shared DB/Redis/MinIO state for a lighter-weight task
ods worktree create codex/fix/ui-polish --dependency-mode shared

# Re-bootstrap an existing worktree
ods worktree bootstrap codex/fix/auth-banner-modal

# Inspect the current worktree's namespaced dependency state
ods worktree deps status

# Reset the current worktree's Postgres/Redis/MinIO namespace
ods worktree deps reset

# See tracked worktrees
ods worktree status

# Show the current worktree manifest
ods worktree show

# Remove a worktree when finished
ods worktree remove codex/fix/auth-banner-modal

# Remove a worktree and tear down its namespaced dependencies
ods worktree remove codex/fix/auth-banner-modal --drop-deps
```

### `verify` - Run the Agent-Lab Verification Ladder

Run a unified verification flow for the current checkout. `ods verify` is the
first worktree-aware entrypoint that combines:

- `agent-check`
- optional targeted pytest execution
- optional targeted Playwright execution
- machine-readable verification summaries written to the worktree artifact dir

```shell
ods verify
```

Useful flags:

| Flag | Description |
|------|-------------|
| `--base-ref <ref>` | Ref to compare against for `agent-check` |
| `--skip-agent-check` | Skip the diff-based rules step |
| `--worktree <id>` | Run verification against a tracked worktree from the control checkout |
| `--pytest <path>` | Run a specific pytest path or node id (repeatable) |
| `--playwright <path>` | Run a specific Playwright test path (repeatable) |
| `--playwright-grep <expr>` | Pass `--grep` through to Playwright |
| `--playwright-project <name>` | Limit Playwright to one project |

Examples:

```shell
# Run just the diff-based checks
ods verify

# Validate a backend change with one focused integration target
ods verify --pytest backend/tests/integration/tests/streaming_endpoints/test_chat_stream.py

# Validate a UI change with one Playwright suite
ods verify --playwright tests/e2e/chat/welcome_page.spec.ts --playwright-project admin

# Run both backend and UI checks
ods verify \
  --pytest backend/tests/integration/tests/streaming_endpoints/test_chat_stream.py \
  --playwright tests/e2e/admin/default-agent.spec.ts

# Verify a tracked product worktree from the control checkout
ods verify --worktree codex/fix/auth-banner-modal
```

### `dev` - Devcontainer Management

Manage the Onyx devcontainer. Also available as `ods dc`.

Requires the [devcontainer CLI](https://github.com/devcontainers/cli) (`npm install -g @devcontainers/cli`).

```shell
ods dev <subcommand>
```

**Subcommands:**

- `up` - Start the devcontainer (pulls the image if needed)
- `into` - Open a zsh shell inside the running devcontainer
- `exec` - Run an arbitrary command inside the devcontainer
- `restart` - Remove and recreate the devcontainer
- `rebuild` - Pull the latest published image and recreate
- `stop` - Stop the running devcontainer

The devcontainer image is published to `onyxdotapp/onyx-devcontainer` and
referenced by tag in `.devcontainer/devcontainer.json` — no local build needed.

**Examples:**

```shell
# Start the devcontainer
ods dev up

# Open a shell
ods dev into

# Run a command
ods dev exec -- npm test

# Restart the container
ods dev restart

# Pull latest image and recreate
ods dev rebuild

# Stop the container
ods dev stop

# Same commands work with the dc alias
ods dc up
ods dc into
```

### `db` - Database Administration

Manage PostgreSQL database dumps, restores, and migrations.

```shell
ods db <subcommand>
```

**Subcommands:**

- `dump` - Create a database dump
- `restore` - Restore from a dump
- `upgrade`/`downgrade` - Run database migrations
- `drop` - Drop a database

Run `ods db --help` for detailed usage.

### `openapi` - OpenAPI Schema Generation

Generate OpenAPI schemas and client code.

```shell
ods openapi all
```

### `check-lazy-imports` - Verify Lazy Import Compliance

Check that specified modules are only lazily imported (used for keeping backend startup fast).

```shell
ods check-lazy-imports
```

### `agent-check` - Check New Agent-Safety Violations

Run a small set of diff-based checks aimed at keeping new changes agent-friendly
without failing on historical debt already present in the repository.

This command is part of the expected workflow on `agent-lab`. It is not necessarily a repo-wide
mandatory gate on `main`.

```shell
ods agent-check
```

Current checks flag newly added:

- `HTTPException` usage in backend product code
- `response_model=` on backend APIs
- Celery `.delay()` calls
- imports from `web/src/components/` outside the legacy component tree

The command also validates the `docs/agent/` knowledge base by checking that
required files exist and that local markdown links in that surface resolve
correctly.

Useful flags:

| Flag | Description |
|------|-------------|
| `--staged` | Check the staged diff instead of the working tree |
| `--base-ref <ref>` | Diff against a git ref other than `HEAD` |
| `--worktree <id>` | Check a tracked worktree from the control checkout |

Examples:

```shell
# Check working tree changes
ods agent-check

# Check only staged changes
ods agent-check --staged

# Compare the branch against main
ods agent-check --base-ref origin/main

# Limit the diff to specific paths
ods agent-check web/src backend/onyx/server/features/build

# Run against a tracked product worktree from the control checkout
ods agent-check --worktree codex/fix/auth-banner-modal --base-ref origin/main
```

### `run-ci` - Run CI on Fork PRs

Pull requests from forks don't automatically trigger GitHub Actions for security reasons.
This command creates a branch and PR in the main repository to run CI on a fork's code.

```shell
ods run-ci <pr-number>
```

**Example:**

```shell
# Run CI for PR #7353 from a fork
ods run-ci 7353
```

### `cherry-pick` - Backport Commits to Release Branches

Cherry-pick one or more commits to release branches and automatically create PRs.
Cherry-pick PRs created by this command are labeled `cherry-pick 🍒`.

```shell
ods cherry-pick <commit-sha> [<commit-sha>...] [--release <version>]
```

**Examples:**

```shell
# Cherry-pick a single commit (auto-detects release version)
ods cherry-pick abc123

# Cherry-pick to a specific release
ods cherry-pick abc123 --release 2.5

# Cherry-pick to multiple releases
ods cherry-pick abc123 --release 2.5 --release 2.6

# Cherry-pick multiple commits
ods cherry-pick abc123 def456 ghi789 --release 2.5
```

### `screenshot-diff` - Visual Regression Testing

Compare Playwright screenshots against baselines and generate visual diff reports.
Baselines are stored per-project and per-revision in S3:

```
s3://<bucket>/baselines/<project>/<rev>/
```

This allows storing baselines for `main`, release branches (`release/2.5`), and
version tags (`v2.0.0`) side-by-side. Revisions containing `/` are sanitised to
`-` in the S3 path (e.g. `release/2.5` → `release-2.5`).

```shell
ods screenshot-diff <subcommand>
```

**Subcommands:**

- `compare` - Compare screenshots against baselines and generate a diff report
- `upload-baselines` - Upload screenshots to S3 as new baselines

The `--project` flag provides sensible defaults so you don't need to specify every path.
When set, the following defaults are applied:

| Flag | Default |
|------|---------|
| `--baseline` | `s3://onyx-playwright-artifacts/baselines/<project>/<rev>/` |
| `--current` | `web/output/screenshots/` |
| `--output` | `web/output/screenshot-diff/<project>/index.html` |
| `--rev` | `main` |

The S3 bucket defaults to `onyx-playwright-artifacts` and can be overridden with the
`PLAYWRIGHT_S3_BUCKET` environment variable.

**`compare` Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | | Project name (e.g. `admin`); sets sensible defaults |
| `--rev` | `main` | Revision baseline to compare against |
| `--from-rev` | | Source (older) revision for cross-revision comparison |
| `--to-rev` | | Target (newer) revision for cross-revision comparison |
| `--baseline` | | Baseline directory or S3 URL (`s3://...`) |
| `--current` | | Current screenshots directory or S3 URL (`s3://...`) |
| `--output` | `screenshot-diff/index.html` | Output path for the HTML report |
| `--threshold` | `0.2` | Per-channel pixel difference threshold (0.0–1.0) |
| `--max-diff-ratio` | `0.01` | Max diff pixel ratio before marking as changed |

**`upload-baselines` Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | | Project name (e.g. `admin`); sets sensible defaults |
| `--rev` | `main` | Revision to store the baseline under |
| `--dir` | | Local directory containing screenshots to upload |
| `--dest` | | S3 destination URL (`s3://...`) |
| `--delete` | `false` | Delete S3 files not present locally |

**Examples:**

```shell
# Compare local screenshots against the main baseline (default)
ods screenshot-diff compare --project admin

# Compare against a release branch baseline
ods screenshot-diff compare --project admin --rev release/2.5

# Compare two revisions directly (both sides fetched from S3)
ods screenshot-diff compare --project admin --from-rev v1.0.0 --to-rev v2.0.0

# Compare with explicit paths
ods screenshot-diff compare \
  --baseline ./baselines \
  --current ./web/output/screenshots/ \
  --output ./report/index.html

# Upload baselines for main (default)
ods screenshot-diff upload-baselines --project admin

# Upload baselines for a release branch
ods screenshot-diff upload-baselines --project admin --rev release/2.5

# Upload baselines for a version tag
ods screenshot-diff upload-baselines --project admin --rev v2.0.0

# Upload with delete (remove old baselines not in current set)
ods screenshot-diff upload-baselines --project admin --delete
```

The `compare` subcommand writes a `summary.json` alongside the report with aggregate
counts (changed, added, removed, unchanged). The HTML report is only generated when
visual differences are detected.

### `trace` - View Playwright Traces from CI

Download Playwright trace artifacts from a GitHub Actions run and open them
with `playwright show-trace`. Traces are only generated for failing tests
(`retain-on-failure`).

```shell
ods trace [run-id-or-url]
```

The run can be specified as a numeric run ID, a full GitHub Actions URL, or
omitted to find the latest Playwright run for the current branch.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--branch`, `-b` | | Find latest run for this branch |
| `--pr` | | Find latest run for this PR number |
| `--project`, `-p` | | Filter to a specific project (`admin`, `exclusive`, `lite`) |
| `--list`, `-l` | `false` | List available traces without opening |
| `--no-open` | `false` | Download traces but don't open them |

When multiple traces are found, an interactive picker lets you select which
traces to open. Use arrow keys or `j`/`k` to navigate, `space` to toggle,
`a` to select all, `n` to deselect all, and `enter` to open. Falls back to a
plain-text prompt when no TTY is available.

Downloaded artifacts are cached in `/tmp/ods-traces/<run-id>/` so repeated
invocations for the same run are instant.

**Examples:**

```shell
# Latest run for the current branch
ods trace

# Specific run ID
ods trace 12345678

# Full GitHub Actions URL
ods trace https://github.com/onyx-dot-app/onyx/actions/runs/12345678

# Latest run for a PR
ods trace --pr 9500

# Latest run for a specific branch
ods trace --branch main

# Only download admin project traces
ods trace --project admin

# List traces without opening
ods trace --list
```

### `journey` - Capture Before/After Browser Journeys

Run a registered Playwright journey with video capture. The default workflow is
to record `before` and `after` inside the same tracked worktree as the change.
`journey compare` remains available as a recovery path when you need to compare
two explicit revisions/worktrees after the fact.

Registered journeys live in `web/tests/e2e/journeys/registry.json`.
An optional `.github/agent-journeys.json` file can list journeys for a PR:

```json
{
  "journeys": ["auth-landing"]
}
```

```shell
ods journey <subcommand>
```

**Subcommands:**

- `list` - Show registered journeys
- `run` - Run one journey against the current or target worktree
- `compare` - Capture `before` and `after` artifacts across two revisions/worktrees when a missed baseline must be recovered
- `publish` - Upload a compare run to S3 and upsert the PR comment

**Examples:**

```shell
# List journey definitions
ods journey list

# Capture before in the tracked product worktree before editing
ods journey run --worktree codex/fix/auth-banner-modal --journey auth-landing --label before

# Capture after in that same worktree after validating the fix
ods journey run --worktree codex/fix/auth-banner-modal --journey auth-landing --label after

# Recover a missed baseline later by comparing origin/main to a tracked product worktree
ods journey compare \
  --journey auth-landing \
  --after-worktree codex/fix/auth-banner-modal

# Publish an existing compare run to PR #10007
ods journey publish \
  --run-dir .git/onyx-agent-lab/journeys/20260408-123000 \
  --pr 10007
```

`journey run` writes a `summary.json` into the capture directory. `journey compare`
writes a `summary.json` into its run directory and, when `--pr` is supplied,
uploads that directory to S3 and upserts a PR comment with before/after links.

### `pr-review` - Fetch and Respond to GitHub Review Threads

Treat PR review comments as a local machine-readable workflow instead of relying
on the GitHub UI alone.

```shell
ods pr-review <subcommand>
```

**Subcommands:**

- `fetch` - Download review threads into local harness state
- `triage` - Classify threads as actionable, duplicate, outdated, or resolved
- `respond` - Reply to an inline review comment and optionally resolve its thread
- `resolve` - Resolve a review thread without posting a reply

**Examples:**

```shell
# Fetch review threads for the current branch PR
ods pr-review fetch

# Triage review threads for a specific PR
ods pr-review triage --pr 10007

# Reply to a top-level review comment and resolve the thread
ods pr-review respond \
  --pr 10007 \
  --comment-id 2512997464 \
  --thread-id PRRT_kwDO... \
  --body "Fixed in the latest patch. Added a regression journey as well."
```

Fetched and triaged review data is written under the local harness state
directory:

```text
$(git rev-parse --git-common-dir)/onyx-agent-lab/reviews/pr-<number>/
```

### `pr-checks` - Diagnose Failing GitHub Checks

Inspect the latest checks on a PR and surface the failing ones with the next
recommended remediation command.

```shell
ods pr-checks <subcommand>
```

**Subcommands:**

- `status` - list all checks for the PR
- `diagnose` - list only failing checks and point to the next step

**Examples:**

```shell
# Show all checks on the current branch PR
ods pr-checks status

# Show only failing checks and the next remediation command
ods pr-checks diagnose --pr 10007
```

`pr-checks diagnose` is especially useful after pushing a fix or after replying
to review comments. For Playwright failures it points directly at `ods trace`.

### `pr-open` - Open a PR With the Repo Template

Create a pull request through `gh` while enforcing a conventional-commit title.
If `--title` is omitted, `ods` uses the latest commit subject. The PR body
defaults to `.github/pull_request_template.md`. PRs are ready-for-review by
default; use `--draft` only when you explicitly need that state.

```shell
ods pr-open
ods pr-open --title "fix: suppress logged-out modal on fresh auth load"
```

### `pr-merge` - Merge a PR Through `gh`

Merge or auto-merge a pull request with an explicit merge method.

```shell
ods pr-merge --pr 10007 --method squash
ods pr-merge --pr 10007 --method squash --auto --delete-branch
```

### Testing Changes Locally (Dry Run)

Both `run-ci` and `cherry-pick` support `--dry-run` to test without making remote changes:

```shell
# See what would happen without pushing
ods run-ci 7353 --dry-run
ods cherry-pick abc123 --release 2.5 --dry-run
```

## Upgrading

To upgrade the stable version, upgrade it as you would any other [requirement](https://github.com/onyx-dot-app/onyx/tree/main/backend/requirements#readme).

## Building from source

Generally, `go build .` or `go install .` are sufficient.

`go build .` will output a `tools/ods/ods` binary which you can call normally,

```shell
./ods --version
```

while `go install .` will output to your [GOPATH](https://go.dev/wiki/SettingGOPATH) (defaults `~/go/bin/ods`),

```shell
~/go/bin/ods --version
```

_Typically, `GOPATH` is added to your shell's `PATH`, but this may be confused easily during development
with the pip version of `ods` installed in the Onyx venv._

To build the wheel,

```shell
uv build --wheel
```

To build and install the wheel,

```shell
uv pip install .
```

## Deploy

Releases are deployed automatically when git tags prefaced with `ods/` are pushed to [GitHub](https://github.com/onyx-dot-app/onyx/tags).

The [release-tag](https://pypi.org/project/release-tag/) package can be used to calculate and push the next tag automatically,

```shell
tag --prefix ods
```

See also, [`.github/workflows/release-devtools.yml`](https://github.com/onyx-dot-app/onyx/blob/main/.github/workflows/release-devtools.yml).
