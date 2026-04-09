# Initial Architecture Map

Status: provisional baseline. This is a routing map for agents, not a complete design spec for
every subsystem. Update it as the repo becomes more explicit.

## Top-Level Surfaces

The repository is easiest to reason about as six main surfaces:

| Surface | Primary Paths | Purpose |
| --- | --- | --- |
| Backend product logic | `backend/onyx/`, `backend/ee/onyx/` | Core auth, chat, search, indexing, connectors, API, and enterprise extensions |
| Data and persistence | `backend/onyx/db/`, `backend/ee/onyx/db/`, `backend/alembic/` | DB models, data access logic, and schema migrations |
| Frontend product surfaces | `web/src/app/`, `web/src/sections/`, `web/src/layouts/` | Next.js routes, screens, and feature-level UI composition |
| Frontend design system and shared UI | `web/lib/opal/`, `web/src/refresh-components/` | Preferred primitives for new UI work |
| Devtools and local developer workflows | `tools/ods/`, `cli/` | Repo automation, CI helpers, visual regression tooling, and CLI integrations |
| Agent-facing platform work | `backend/onyx/server/features/build/`, `backend/onyx/mcp_server/`, `backend/onyx/deep_research/`, `backend/onyx/agents/` | Sandbox runtime, MCP tool surface, agent orchestration, and research workflows |
| Agent-lab harness state | shared git metadata under `$(git rev-parse --git-common-dir)/onyx-agent-lab/` | Local worktree manifests, ports, env overlays, and verification artifacts for agentized development |

## Backend Map

Use these paths as the first stop when routing backend changes:

| Area | Paths | Notes |
| --- | --- | --- |
| Authentication and access control | `backend/onyx/auth/`, `backend/onyx/access/`, `backend/ee/onyx/access/` | User identity, auth flows, permissions |
| Chat and answer generation | `backend/onyx/chat/`, `backend/onyx/server/query_and_chat/` | Chat loop, message processing, streaming |
| Retrieval and tools | `backend/onyx/tools/`, `backend/onyx/context/`, `backend/onyx/mcp_server/` | Search tools, web tools, context assembly, MCP exposure |
| Connectors and indexing | `backend/onyx/connectors/`, `backend/onyx/document_index/`, `backend/onyx/background/` | Source sync, indexing, pruning, permissions sync |
| LLM and prompt infrastructure | `backend/onyx/llm/`, `backend/onyx/prompts/`, `backend/ee/onyx/prompts/` | Provider integrations and prompting |
| Server APIs and feature entrypoints | `backend/onyx/server/`, `backend/ee/onyx/server/` | FastAPI routes and product feature APIs |
| Agent and build platform | `backend/onyx/server/features/build/`, `backend/onyx/agents/`, `backend/onyx/deep_research/` | Sandboxes, agent runtimes, orchestration, long-running research |
| Persistence | `backend/onyx/db/`, `backend/ee/onyx/db/` | Put DB operations here, not in route handlers or feature modules |

## Frontend Map

For frontend work, route changes by intent first, then by component maturity:

| Intent | Preferred Paths | Notes |
| --- | --- | --- |
| Next.js route/page work | `web/src/app/` | App Router pages and page-local wiring |
| Feature composition | `web/src/sections/`, `web/src/layouts/` | Preferred place for reusable feature-level assemblies |
| New shared UI primitives | `web/lib/opal/`, `web/src/refresh-components/` | Default targets for new reusable UI |
| Legacy shared UI | `web/src/components/` | Avoid for new work unless forced by the local surface |
| Frontend business logic | `web/src/lib/`, `web/src/hooks/`, `web/src/interfaces/` | Utilities, hooks, typed interfaces |

Important frontend rule already established in [web/AGENTS.md](../../web/AGENTS.md):

- Do not use `web/src/components/` for new component work.

## Existing Hard Constraints

These rules already exist and should be treated as architectural boundaries:

- Backend errors should raise `OnyxError`, not `HTTPException`.
- DB operations belong under `backend/onyx/db/` or `backend/ee/onyx/db/`.
- New FastAPI APIs should not use `response_model`.
- Celery tasks should use `@shared_task`.
- Enqueued Celery tasks must include `expires=`.
- Backend calls in local/manual flows should go through `http://localhost:3000/api/...`.

## Change Routing Heuristics

Use these heuristics before editing:

1. If the task changes persistence semantics, start in the DB layer and migrations.
2. If the task changes user-visible UI, find the route in `web/src/app/`, then move downward into
   `sections`, `layouts`, and preferred shared UI.
3. If the task spans product behavior and background execution, inspect both the API entrypoint and
   the relevant Celery path.
4. If the task concerns agentization, build, or local execution, check whether
   `backend/onyx/server/features/build/` or `tools/ods/` is the better home before creating a new
   subsystem.
5. If the task needs isolated local boot, browser validation, or per-change artifacts, check
   [HARNESS.md](./HARNESS.md) before inventing another ad hoc runner.
6. If the change touches a historically messy area, consult [LEGACY_ZONES.md](./LEGACY_ZONES.md)
   before adding more local patterns.

## Test Routing

Onyx already has a clear testing ladder:

- `backend/tests/unit/`: isolated logic only
- `backend/tests/external_dependency_unit/`: real infra, direct function calls, selective mocking
- `backend/tests/integration/`: real deployment, no mocking
- `web/tests/e2e/`: full frontend-backend coordination

Prefer the lowest layer that still validates the real behavior. For many product changes in this
repo, that means integration or Playwright rather than unit tests.
