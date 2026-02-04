## Purpose
Instructions for agents working in this repository.

## Long-Run Autonomy Mode
When the user explicitly asks for a long, uninterrupted build (8+ hours):
- Proceed without asking additional questions unless a hard blocker prevents progress.
- Make reasonable defaults and record them in `DECISIONS.md`.
- Keep work in small, coherent commits or checkpoints and note status in `STATUS.md`.
- Run tests and basic verification before stopping; if failures remain, document them in `STATUS.md`.
- Keep the repo runnable at each major step.
- If you hit something that truly requires user input or credentials, record it in `blocker.md` and continue with the next best parallel task.

## Current State
- Early development; only `README.md` and `Web-Crawler-Go_Basic_System_Ideation.jpeg` are present.
- Do not invent structure. Align any additions with `README.md`.

## Planned Layout (Not Yet Created)
- `/backend` — Go API + crawler core.
- `/frontend` — Next.js + TypeScript + React.
- `/infra` — Docker Compose and supporting infra.

## Tooling & Commands
- Frontend: use Bun once `/frontend` exists. Prefer `bun run <script>` from `package.json`.
- Backend: use standard Go tooling once `/backend` exists; prefer `go test ./...` and `go vet ./...`.
- If scripts/configs are missing, do not invent commands; update `README.md` when adding them.

## One-Shot Reference Docs
Use and maintain these docs when implementing:
- `PROJECT_SPEC.md` for scope and success criteria.
- `ARCHITECTURE.md` for components and data flow.
- `API.md` for service endpoints and payloads.
- `SCHEMA.md` for Postgres tables and indices.
- `FRONTEND_SPEC.md` for UI behavior and live updates.
- `IMPLEMENTATION_PLAN.md` for the step-by-step build plan.
- `TEST_PLAN.md` for required tests and scenarios.
- `DECISIONS.md` for defaults and tradeoffs.
- `STATUS.md` for current progress and blockers.
- `blocker.md` for anything that needs user input; keep all blockers in one place.

## Change Hygiene
- Update `README.md` when adding new top-level directories or developer commands.
- Avoid adding new package managers (no npm/yarn if Bun is set).

## Decision Guardrails
- If a change would alter the high-level architecture, call it out for confirmation first.
