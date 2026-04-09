# Quality Score Baseline

This file is an intentionally rough baseline for how legible the repository is to coding agents.
It is not a product quality report. It is a scorecard for agent development ergonomics.

## Scoring Rubric

Each area is scored from `0` to `5` on four dimensions:

- `Legibility`: how easy it is to discover the right files and concepts
- `Boundaries`: how clearly dependency and ownership seams are defined
- `Verification`: how available and reliable the feedback loops are
- `Agent ergonomics`: how likely an agent is to make a correct change without human rescue

Overall score is directional, not mathematically precise.

## Initial Baseline

| Area | Legibility | Boundaries | Verification | Agent ergonomics | Overall | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Backend core (`backend/onyx/`, `backend/ee/onyx/`) | 3 | 3 | 4 | 3 | 3.25 | Strong test surface, but top-level routing docs are thin |
| Persistence (`backend/onyx/db/`, migrations) | 4 | 4 | 3 | 4 | 3.75 | Clearer than most areas because path-level rules already exist |
| Frontend modern surfaces (`web/src/app/`, `sections`, `opal`, `refresh-components`) | 3 | 3 | 3 | 3 | 3.0 | Direction exists, but mixed generations still leak across boundaries |
| Frontend legacy shared UI (`web/src/components/`) | 1 | 1 | 2 | 1 | 1.25 | Explicitly deprecated, but still present and easy for agents to cargo-cult |
| Agent platform and build sandbox (`backend/onyx/server/features/build/`) | 3 | 4 | 3 | 4 | 3.5 | Good substrate for agentization, but not yet aimed at repo development workflows |
| MCP, CLI, and devtools (`backend/onyx/mcp_server/`, `cli/`, `tools/ods/`) | 4 | 4 | 4 | 4 | 4.0 | `agent-check`, worktree manifests, `ods verify`, `ods journey`, and PR review/check tooling give this surface a real control plane |
| Repo-level docs and plans | 4 | 3 | 4 | 4 | 3.75 | `docs/agent/` now describes the journey/review/check loop directly, though subsystem coverage is still uneven |

## Biggest Gaps

1. Repo-level architecture knowledge is still thinner than the runtime and workflow docs.
2. Brownfield and legacy zones are not explicitly flagged enough for agents.
3. Important engineering rules still outnumber the mechanical checks that enforce them.
4. The worktree harness does not yet include a local observability stack or one-command process orchestration.

## Near-Term Targets

The next improvements should aim to move these areas:

- Repo-level docs and plans: `3.0 -> 4.0`
- Frontend legacy safety: `1.25 -> 2.5`
- Backend core agent ergonomics: `3.0 -> 4.0`
- Worktree observability and runtime automation: `2.5 -> 4.0`

## Update Policy

When a new check, map, or workflow materially improves agent behavior, update this scorecard and
note what changed. If a score changes, the adjacent notes should explain why.
