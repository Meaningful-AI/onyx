# Legacy Zones

Status: initial classification. This file exists to stop agents from treating every existing
pattern in the repository as equally desirable precedent.

## Zone Types

| Zone | Meaning | Edit Policy |
| --- | --- | --- |
| `strict` | Preferred surface for new work | Freely extend, but keep boundaries explicit and add tests |
| `transition` | Actively evolving surface with mixed patterns | Prefer local consistency, avoid introducing new abstractions casually |
| `legacy-adapter` | Known historical surface or deprecated pattern area | Avoid new dependencies on it; prefer facades, wrappers, or migrations away |
| `frozen` | Only touch for bug fixes, security, or explicitly scoped work | Do not expand the pattern set |

## Initial Classification

### Strict

These are good default targets for new investment:

- `backend/onyx/db/`
- `backend/ee/onyx/db/`
- `backend/onyx/error_handling/`
- `backend/onyx/mcp_server/`
- `backend/onyx/server/features/build/`
- `tools/ods/`
- `web/lib/opal/`
- `web/src/refresh-components/`
- `web/src/layouts/`
- `web/src/sections/cards/`

### Transition

These areas are important and active, but they mix styles, eras, and responsibilities:

- `backend/onyx/server/`
- `backend/ee/onyx/server/`
- `backend/onyx/chat/`
- `backend/onyx/tools/`
- `backend/onyx/agents/`
- `backend/onyx/deep_research/`
- `web/src/app/`
- `web/src/sections/`
- `web/src/lib/`

Edit guidance:

- prefer incremental refactors over sweeping rewrites
- keep changes local when the area lacks clear boundaries
- add tests before extracting new shared abstractions

### Legacy-Adapter

These areas should not be treated as default precedent for new work:

- `web/src/components/`
- `backend/model_server/legacy/`

Edit guidance:

- do not add fresh reusable components or helper patterns here
- if a task requires touching these areas, prefer introducing an adapter in a stricter surface
- if you must extend a legacy file, keep the blast radius small and document follow-up cleanup

### Frozen

No repo-wide frozen zones are declared yet beyond files or subsystems that are clearly deprecated on
their face. Add explicit entries here rather than relying on tribal knowledge.

## Brownfield Rules

When a task lands in a non-strict zone:

1. Identify whether the task is fixing behavior, adding capability, or migrating structure.
2. Avoid copying local patterns into stricter parts of the codebase.
3. If an unsafe pattern is unavoidable, isolate it behind a typed boundary.
4. Record newly discovered smells in [GOLDEN_RULES.md](./GOLDEN_RULES.md) or a follow-on
   execution plan.

## Promotion Criteria

A transition area can move toward `strict` when:

- its dependency boundaries are easy to explain
- new code has a preferred home
- tests are reliable enough for agents to use as feedback loops
- recurring review comments have been turned into written or mechanical rules
