# Agent Engineering Docs

This directory is the knowledge base for the `agent-lab` workflow around making development of
`onyx` itself more agentized.

The goal is not to replace the root [AGENTS.md](../../AGENTS.md).
The goal is to keep architecture maps, unsafe-zone notes, quality signals, and follow-on
execution plans in a form that coding agents can discover and update.

On `agent-lab`, this directory is the system of record for agent-engineering workflow.

## Principles

- Keep the entrypoint small. The root `AGENTS.md` should point here; it should not become a
  growing encyclopedia.
- Create the target worktree first. The intended workflow is one task, one tracked worktree, one
  verification loop, and one PR from that same checkout.
- Keep artifacts with the workflow. Browser videos, traces, review summaries, and check triage
  should be produced by harness commands and stored as machine-readable outputs, not recreated
  from chat memory.
- Prefer maps over manuals. Agents need navigable pointers to the right subsystem, not a giant
  blob of undifferentiated instructions.
- Encode recurring judgment into the repo. If a rule matters often, document it here and then
  promote it into a check, linter, test, or script.
- Distinguish legacy from greenfield. Agents will copy the patterns they see. If an area is
  historically messy, we need to say so explicitly.
- Version decisions with the code. If a design choice matters for future changes, it should live
  in-repo rather than in chat or memory.

## Documents

- [ARCHITECTURE.md](./ARCHITECTURE.md): top-level codebase map and change-routing guidance.
- [BRANCHING.md](./BRANCHING.md): branch model for long-running `agent-lab` development and
  promotion of product-only changes to `main`.
- [HARNESS.md](./HARNESS.md): worktree runtime model, verification ladder, and browser/tooling
  expectations.
- [LEGACY_ZONES.md](./LEGACY_ZONES.md): edit policy for strict, transitional, and legacy areas.
- [GOLDEN_RULES.md](./GOLDEN_RULES.md): active rules for `agent-lab` and promotion targets for
  mechanical enforcement.
- [QUALITY_SCORE.md](./QUALITY_SCORE.md): baseline legibility and maintainability assessment for
  agent work.

## Operating Model

Use this directory for information that should change how future agents work in the `agent-lab`
workflow:

- architecture maps
- dependency and layering rules
- "do not extend this pattern" warnings
- safe extension points
- recurring cleanup policies
- harness/runtime behavior for worktree-based development
- before/after browser journeys and PR artifact publication
- GitHub review and failing-check control loops
- quality scorecards
- active execution plans for agent-engineering improvements

Current workflow split:

- `codex/agent-lab` is the control checkout for the harness itself.
- `codex/lab/<name>` branches are for harness/docs/tooling work based on `codex/agent-lab`.
- `codex/fix/<name>`, `codex/feat/<name>`, and similar conventional product branches should be
  created from `origin/main`, even when they are managed from the `agent-lab` control checkout.
- PR titles and commit messages should use conventional-commit style, never `[codex]` prefixes.

Do not turn this into a dumping ground. If something is local to one feature, keep it with that
feature. This directory is for `agent-lab`-level agent-development guidance.
