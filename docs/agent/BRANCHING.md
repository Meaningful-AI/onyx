# Branching Model for `agent-lab`

This is the branching policy for `agent-lab`. It is intentionally separate from the default
workflow on `main`.

This document explains how to use a long-running `agent-lab` branch without making `main`
implicitly depend on lab-only agent-engineering changes.

## Goals

- Keep `main` stable and consensus-driven.
- Allow opt-in agent-engineering improvements to live on `agent-lab`.
- Let engineers and agents use `agent-lab` as a control checkout for worktree-based development.
- Ensure product PRs to `main` originate from `main`-based branches, not from `agent-lab`.

## Branch Roles

| Branch | Purpose |
| --- | --- |
| `main` | Shipping branch and team default |
| `codex/agent-lab` | Long-running control checkout containing the harness and agent-engineering improvements |
| `codex/lab/<name>` | Short-lived branch for `agent-lab`-only tooling, docs, or workflow work |
| `codex/fix/<name>`, `codex/feat/<name>`, etc. | Short-lived product branch cut from `origin/main` and managed by the `agent-lab` control checkout |

## Core Rule

`main` must never depend on `agent-lab`.

That means:

- `codex/agent-lab` may contain extra tooling, docs, checks, and workflow changes.
- Product branches may be managed by the `agent-lab` control checkout, but they must still be based
  on `origin/main`.
- A PR to `main` should come from a `main`-based product branch, not from `codex/agent-lab`.

## Preferred Workflow

### Lab-Only Work

Use this for agent-engineering docs, harnesses, optional checks, or tooling that should remain on
`agent-lab` for now.

1. Branch from `codex/agent-lab` into `codex/lab/<name>`.
   For local isolation, create the branch via `ods worktree create codex/lab/<name>`.
2. Make the lab-only changes.
3. Open the PR back into `codex/agent-lab`.
4. Do not open these changes directly to `main` unless the team later agrees to upstream them.

### Product Feature Work

Use this when you want to fix a product bug or build a shipping feature for `main`.

1. Stay in the `codex/agent-lab` control checkout.
2. Create a product worktree from `origin/main`, using a conventional branch lane such as:
   - `ods worktree create codex/fix/<name>`
   - `ods worktree create codex/feat/<name>`
3. Make the code changes inside that worktree checkout.
4. Run harness commands from the control checkout against the tracked worktree:
   - `ods agent-check --worktree codex/fix/<name>`
   - `ods verify --worktree codex/fix/<name>`
   - `ods backend api --worktree codex/fix/<name>`
   - `ods web dev --worktree codex/fix/<name>`
5. If the change needs browser proof, record a before/after journey:
   - before editing: `ods journey run --worktree codex/fix/<name> --journey <name> --label before`
   - after validating the fix: `ods journey run --worktree codex/fix/<name> --journey <name> --label after`
   - use `ods journey compare` only when the initial `before` capture was missed and a recovery
     baseline is needed later
   - after the PR exists, publish the artifact directory you captured or the fallback compare run
     with `ods journey publish --run-dir <dir> --pr <number>`
6. Commit, push, and open the PR from the product worktree checkout itself.
   Prefer `ods pr-open` so the repo template and conventional-commit title check stay in the same
   control plane.
7. Open the PR directly from that product branch to `main`.
8. After the PR is open, use:
   - `ods pr-review triage --pr <number>`
   - `ods pr-checks diagnose --pr <number>`
   - `ods pr-review respond --comment-id ... --thread-id ... --body ...`

## Commit Hygiene Rules

This workflow only works if commits are separated cleanly.

Agents and humans should:

- keep lab-only workflow changes in separate commits from product logic
- avoid mixing refactors, harness changes, and feature behavior in one commit
- use conventional-commit messages and PR titles
- prefer multiple small commits over one large mixed commit

Good split:

- `docs(agent-lab): clarify control-checkout workflow`
- `fix: suppress logged-out modal on fresh unauthenticated load`
- `test: add regression coverage for auth-page logout modal`

Bad split:

- `misc: update agent docs, add lint, change connector UI, fix API`

## Guidance for Agents

When an agent is working on product code, it should assume:

1. The product branch should be created from `origin/main`, not from `codex/agent-lab`.
2. The `codex/agent-lab` checkout is the control plane for `ods` commands until the harness is
   upstreamed more broadly.
3. The code change itself should still be made and committed inside the target product worktree.
4. A PR to `main` should use a conventional-commit title such as `fix: ...` or `feat: ...`.

If a product bug is discovered while editing on `codex/agent-lab`, treat that as exploration.
Restart the real fix in a fresh `main`-based product worktree and port only the minimal product
patch there.

## What Should Usually Stay on `agent-lab`

These are usually lab-only unless explicitly approved for upstreaming:

- branch-specific workflow docs
- harness-only `ods` commands
- non-consensus lint rules
- agent harness scripts
- opt-in automation for review or promotion
- branch-specific AGENTS guidance

## What Can Be Promoted to `main`

These can be promoted once they stand on their own:

- product feature code
- product tests
- bug fixes
- low-controversy lint rules with team agreement
- small devtools improvements that are useful outside `agent-lab`

## Review Standard

If opening a PR to `main` from the `agent-lab` control workflow:

- make sure the PR branch itself is based on `origin/main`
- use a conventional-commit title
- mention any control-plane validation that was run with `ods ... --worktree <branch>`
- attach journey artifacts when browser behavior changed
- treat review-thread replies and failing checks as part of the same agent loop, not as a separate
  manual phase

This keeps the product branch reviewable without forcing reviewers to understand the entire
`agent-lab` branch.
