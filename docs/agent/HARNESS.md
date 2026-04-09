# Worktree Harness

This document defines the `agent-lab` harness model for doing end-to-end work on `onyx`.

The goal is to make one agent capable of taking one isolated change from edit to verification
without depending on human memory for ports, paths, or validation steps.

## Principles

These decisions follow the same principles described in OpenAI's
[Harness engineering](https://openai.com/index/harness-engineering/) and
[Unlocking the Codex harness](https://openai.com/index/unlocking-the-codex-harness/) articles:

- each task should run in its own git worktree
- the app should be bootable per worktree
- browser state should be directly legible to the agent
- logs, traces, and test artifacts should be attached to the same worktree lifecycle
- repository docs plus local metadata should be the system of record, not chat memory

## Current Harness Surface

The first `agent-lab` harness layer lives in `tools/ods/`.

Implemented command surfaces:

- `ods worktree create <branch>`: creates a git worktree plus local agent metadata
- `ods worktree deps up|status|reset|down`: provisions and manages namespaced external state
- `ods worktree status`: lists tracked worktrees and their URLs
- `ods worktree show [worktree]`: prints the manifest for one worktree
- `ods worktree remove <worktree>`: removes the worktree and local harness state
- `ods journey list|run|compare|publish`: records registered browser journeys, including local
  before/after video artifacts and optional PR publication
- `ods pr-review fetch|triage|respond|resolve`: turns GitHub review threads into a local
  machine-readable loop
- `ods pr-checks status|diagnose`: makes failing GitHub checks queryable from the same control
  plane
- `ods verify`: runs the agent verification ladder and writes a machine-readable summary
- `ods agent-check`: runs diff-based architectural and doc checks

## Required Workflow

This is the required `agent-lab` workflow going forward:

1. Create the target worktree first with `ods worktree create`.
2. Make the code changes inside that worktree.
3. Run verification against that same worktree.
4. Open the PR from that same worktree.

Do not implement a change in one checkout and then rsync or patch it into another checkout just to
test it. That is only acceptable when explicitly debugging the harness itself.

Also do not use raw `git worktree add` for harness-managed work. `ods worktree create` is the
authoritative entrypoint because it disables repo hooks during checkout, writes the local manifest,
bootstraps env/runtime dependencies, provisions namespaced state, and records the worktree lane and
base ref.

## Control Checkout Model

Right now the harness code itself lives on `codex/agent-lab`, not on plain `main`.

That means the `codex/agent-lab` checkout acts as the control plane:

- lab worktrees such as `codex/lab/...` are based on `codex/agent-lab`
- product worktrees such as `codex/fix/...` or `codex/feat/...` are based on `origin/main`
- the `agent-lab` checkout can still manage those product worktrees via `--worktree`
  flags on `ods backend`, `ods web`, `ods verify`, and `ods agent-check`

This lets us use the harness to manage a `main`-based product branch before the harness itself has
been upstreamed to `main`.

## Worktree Metadata

Each `agent-lab` worktree gets a local manifest stored under the shared git metadata directory:

```text
$(git rev-parse --git-common-dir)/onyx-agent-lab/worktrees/<id>/
```

The manifest tracks:

- branch name
- checkout path
- base ref used when the branch was created
- dependency mode and namespace-derived external dependency settings
- reserved ports for web, API, model server, and MCP
- browser-facing URLs
- generated env overlay file paths
- artifact directory
- last verification summary

This state is local runtime metadata. It is intentionally not checked into the repo.

## Boot Model

The current harness boot model isolates the mutable application processes and can also isolate the
mutable non-search data plane.

Per worktree:

- Next.js dev server gets its own `PORT`
- browser-facing base URL is unique
- backend API port is unique
- model server port is unique
- MCP port reservation exists for future worktree-local MCP runtime use
- artifacts are written to a worktree-specific directory

Today this is enough to make the app bootable per worktree without requiring a fully duplicated
dependency container stack for every task.

Important boundary:

- isolated today: app processes, ports, URLs, local artifacts, worktree-local dependency installs,
  PostgreSQL database, Redis key prefix, and MinIO file-store bucket when the worktree runs in
  `namespaced` dependency mode
- shared today: OpenSearch/Vespa and the rest of the local dependency stack started via docker
  compose

This means a normal `agent-lab` worktree can run against:

- a dedicated Postgres database on the shared local Postgres server
- a dedicated Redis namespace on the shared local Redis instance
- a dedicated MinIO file-store bucket on the shared local object store

OpenSearch/Vespa remain shared-only by design on this branch. The harness should never imply
otherwise.

This is a deliberate brownfield adaptation of the OpenAI article’s worktree-per-task model:
keep the common path mechanically isolated where the repo already supports it, and explicitly mark
the high-complexity surfaces that remain shared.

## Dependency Modes

`agent-lab` currently supports two dependency modes:

- `namespaced`: default mode for agent feature work. Creates one Postgres database, one Redis
  prefix, and one MinIO bucket per worktree.
- `shared`: reuse the existing local DB/Redis/MinIO state when full isolation is unnecessary.

The worktree manifest is the source of truth for the selected mode and the derived namespace values.

Search infrastructure policy:

- OpenSearch/Vespa are always shared
- there is no current plan to add namespaced or per-worktree search stacks on `agent-lab`
- tasks that mutate search/index infrastructure should be treated as higher-risk and validated with
  extra care because the harness does not isolate that surface

## Backend and Web Integration

When `ods backend ...` or `ods web ...` runs inside a tracked `agent-lab` worktree, it should
derive runtime settings from the worktree manifest automatically.

Current behavior:

- `ods backend api` defaults to the reserved worktree API port
- `ods backend model_server` defaults to the reserved worktree model-server port
- `ods web dev` gets the reserved worktree web port plus `BASE_URL`, `WEB_DOMAIN`,
  `INTERNAL_URL`, and `MCP_INTERNAL_URL`
- backend and web commands also inherit the manifest’s dependency namespace env overrides
- generated `.vscode/.env.agent-lab` and `.vscode/.env.web.agent-lab` files mirror those values
- `ods worktree bootstrap` prepares the worktree to run by linking env files, linking or cloning
  the Python runtime, and preparing `web/node_modules`
- `ods worktree deps up` provisions namespaced Postgres/Redis/MinIO state when needed
- `ods backend ... --worktree <id>` and `ods web ... --worktree <id>` let the `agent-lab`
  control checkout run app processes against a tracked target worktree

This makes the standard dev commands work in an isolated way without inventing a second startup
surface just for agents.

## Browser Validation

Use two browser surfaces with different jobs:

- Chrome DevTools MCP for exploratory validation, DOM snapshots, navigation, and interactive bug
  reproduction
- Playwright for codified end-to-end verification, screenshots, and retained traces
- `ods journey run` for the default article-style loop inside one worktree: capture `before` before
  the fix, then capture `after` after the fix and publish the resulting artifacts to the PR when
  needed
- `ods journey compare` as the fallback path when the agent missed the initial `before` capture or
  needs a strict baseline-vs-branch comparison after the fact

Important detail:

- The default path should not launch two worktrees just to prove a normal UI bug fix. Use one
  tracked product worktree, start the app in that worktree, and record `before` and `after` from
  that same environment.
- If the fix is still uncommitted, always capture from the tracked target worktree, not from a
  temporary `HEAD` checkout.
- `ods journey compare` is reserved for recovery or explicit revision comparison, not as the
  standard path for every PR.

The worktree manifest's `web` URL is the source of truth for both.

If an agent needs to inspect live UI behavior while iterating, it should prefer Chrome DevTools MCP
against the worktree URL. If the behavior needs to become a repeatable regression check, encode it
as Playwright coverage under `web/tests/e2e/`.

## Verification Ladder

The expected verification sequence for a worktree is:

1. `ods agent-check`
2. targeted backend tests when backend behavior changed
3. targeted Playwright runs when UI or frontend-backend flows changed
4. `ods journey run --label before` before the code change, then `ods journey run --label after`
   after the change when the PR needs durable browser proof
5. screenshot and trace review when UI validation fails

`ods verify` is the first unified entrypoint for this ladder. It writes a JSON summary into the
worktree artifact directory so later agent runs can inspect prior results directly.

For product worktrees based on `main`, the intended control-plane usage is:

1. from `codex/agent-lab`, run `ods worktree create codex/fix/<name>`
2. edit inside the created `main`-based checkout
3. from `codex/agent-lab`, run `ods verify --worktree codex/fix/<name>`
4. if live processes are needed, run `ods backend ... --worktree codex/fix/<name>` and
   `ods web ... --worktree codex/fix/<name>`
5. commit, push, and open the PR from the product worktree checkout itself

## Artifacts

Per-worktree artifacts are written under the local harness state directory, not into chat.

Current artifact classes:

- verification summaries
- pytest logs
- Playwright logs
- journey screenshots, videos, traces, and compare summaries
- PR review thread snapshots and triage outputs
- dependency namespace metadata in the local manifest

Existing repo outputs are still relevant:

- Playwright traces and screenshots under `web/output/`
- screenshot diff reports from `ods screenshot-diff`
- CI trace retrieval from `ods trace`

## Known Gaps

This is the initial harness layer, not the finished system.

Still missing:

- one-command `up/down` orchestration for all local processes
- worktree-local observability stack for logs, metrics, and traces
- worktree-local MCP server runtime wiring
- automatic promotion tooling from `agent-lab` feature branches to `main`
- recurring doc-gardening and cleanup agents
- resumable long-running task server for local development tasks

Resolved in the current harness layer:

- fresh-worktree bootstrap for `.venv`, `.vscode/.env*`, and `web/node_modules`
- namespaced isolation for Postgres, Redis, and MinIO on a per-worktree basis
- registered before/after browser journeys with durable artifact directories
- GitHub review-thread fetch/triage/respond tooling
- GitHub failing-check diagnosis from the same `ods` control plane

Non-goals on this branch:

- OpenSearch/Vespa namespacing
- per-worktree vector/search stacks

Those are the next places to invest if we want to match the article more closely.
