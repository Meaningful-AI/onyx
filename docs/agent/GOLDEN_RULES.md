# Golden Rules

These are the current rules for the `agent-lab` workflow. The long-term goal is to move the useful
ones from prose into shared checks, scripts, or tests where appropriate.

Some of these are already documented elsewhere in the repo as project standards. In this file,
they should be treated as the active rules for work done on `agent-lab`.

## Current Rules

### Backend

1. Raise `OnyxError` instead of `HTTPException`.
2. Put DB operations under `backend/onyx/db/` or `backend/ee/onyx/db/`.
3. Use `@shared_task` for Celery tasks.
4. Never enqueue a Celery task without `expires=`.
5. Do not use FastAPI `response_model` on new APIs.
6. Keep Python strictly typed.

### Frontend

1. Prefer `web/lib/opal/` and `web/src/refresh-components/` for new shared UI.
2. Do not add new shared components under `web/src/components/`.
3. Route backend calls through the frontend `/api/...` surface in local and test flows.
4. Keep TypeScript strictly typed.

### Workflow

1. Start in a tracked worktree created by `ods worktree create`. Do not use raw `git worktree add`
   for harness-managed work.
2. For harness work, use `codex/lab/...` branches based on `codex/agent-lab`. For product work,
   use conventional branches such as `codex/fix/...` or `codex/feat/...` based on `origin/main`.
3. Make edits inside the target worktree. Copying a patch from another checkout is only acceptable
   when debugging the harness itself.
4. Prefer integration or external-dependency-unit tests over unit tests when validating real Onyx
   behavior.
5. When a repeated review comment appears, convert it into repo-local documentation or a mechanical
   check.
6. For browser-visible changes, prefer a registered `ods journey` capture over an ad hoc manual
   recording. The before/after artifacts should live with the PR loop.
7. Use `ods pr-review` to fetch and triage GitHub review threads instead of relying on memory or
   the web UI alone. Reply and resolve from the same workflow when confidence is high.
8. Use `ods pr-checks diagnose` to detect failing GitHub checks and point the next remediation
   command. For Playwright failures, pair it with `ods trace`.
6. PR titles and commit messages should use conventional-commit style such as `fix: ...` or
   `feat: ...`. Never use `[codex]` prefixes in this repo.
9. When touching legacy areas, leave the area more explicit than you found it: better naming,
   better boundaries, or a follow-up cleanup note.

## Mechanical Checks

These are strong candidates for `ods agent-check` or dedicated linters:

| Check | Why it matters |
| --- | --- |
| Ban `HTTPException` in backend product code | Keeps API error handling consistent |
| Ban direct DB mutations outside DB directories | Preserves layering |
| Detect task enqueue calls missing `expires=` | Prevents queue growth and stale work |
| Detect new imports from `web/src/components/` in non-legacy code | Prevents further UI drift |
| Detect direct calls to backend ports in tests/scripts where frontend proxy should be used | Preserves realistic request paths |
| Detect missing docs/agent references for new repo-level rules | Prevents knowledge from staying only in chat |

## Rule Promotion Policy

Promote a rule from prose into enforcement when at least one is true:

- it has been violated more than once
- a violation is expensive to detect late
- the remediation is mechanical
- the error message can teach the correct pattern succinctly

Agents work better with fast, local, actionable failures than with broad stylistic feedback after a
PR is opened.
