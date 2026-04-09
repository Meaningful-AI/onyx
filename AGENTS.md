# Project Knowledge Base

This file is the entrypoint for agents working in this repository. Keep it small.

## Start Here

- General development workflow and repo conventions: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Frontend standards for `web/` and `desktop/`: [web/AGENTS.md](./web/AGENTS.md)
- Backend testing strategy and commands: [backend/tests/README.md](./backend/tests/README.md)
- Celery worker and task guidance: [backend/onyx/background/celery/README.md](./backend/onyx/background/celery/README.md)
- Backend API error-handling rules: [backend/onyx/error_handling/README.md](./backend/onyx/error_handling/README.md)
- Plan-writing guidance: [plans/README.md](./plans/README.md)

## Agent-Lab Docs

When working on `agent-lab` or on tasks explicitly about agent-engineering, use:

- [docs/agent/README.md](./docs/agent/README.md)

These docs are the system of record for the `agent-lab` workflow.

## Universal Notes

- For non-trivial work, create the target worktree first and keep the edit, test, and PR loop
  inside that worktree. Do not prototype in one checkout and copy the patch into another unless
  you are explicitly debugging the harness itself.
- Use `ods worktree create` for harness-managed worktrees. Do not use raw `git worktree add` when
  you want the `agent-lab` workflow, because it will skip the manifest, env overlays, dependency
  bootstrap, and lane-aware base-ref selection.
- When a change needs browser proof, use the harness journey flow instead of ad hoc screen capture:
  record `before` in the target worktree before making the change, then record `after` in that
  same worktree after validation. Use `ods journey compare` only when you need to recover a missed
  baseline or compare two explicit revisions after the fact.
- After opening a PR, treat review feedback and failing checks as part of the same loop:
  use `ods pr-review ...` for GitHub review threads and `ods pr-checks diagnose` plus `ods trace`
  for failing Playwright runs.
- PR titles and commit messages should use conventional-commit style such as `fix: ...` or
  `feat: ...`. Do not use `[codex]` prefixes in this repo.
- If Python dependencies appear missing, activate the root venv with `source .venv/bin/activate`.
- To make tests work, check the root `.env` file for an OpenAI key.
- If using Playwright to explore the frontend, you can usually log in with username `a@example.com`
  and password `a` at `http://localhost:3000`.
- Assume Onyx services are already running unless the task indicates otherwise. Check `backend/log`
  if you need to verify service activity.
- When making backend calls in local development flows, go through the frontend proxy:
  `http://localhost:3000/api/...`, not `http://localhost:8080/...`.
- Put DB operations under `backend/onyx/db/` or `backend/ee/onyx/db/`. Do not add ad hoc DB access
  elsewhere.

## How To Use This File

- Use this file as a map, not a manual.
- Follow the nearest authoritative doc for the subsystem you are changing.
- If a repeated rule matters enough to teach every future agent, document it near the code it
  governs or encode it mechanically.
