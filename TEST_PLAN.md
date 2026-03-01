# Test Plan

## Unit Tests
- URL canonicalization and normalization.
- Dedup behavior (including empty key handling).
- Run config validation/clamping against launch caps.
- Target policy checks (hostname and private/reserved IP blocking).
- Rate limiter logic (`allow`, `retry-after`, window reset).
- Scheduler fairness and per-host limit behavior.
- Retry and circuit-breaker transitions.

## Integration Tests
- `POST /runs` returns structured validation errors for capped fields.
- `POST /runs` rate limiting returns `429` after limit.
- Create/start/stop lifecycle with valid requests.
- SSE stream behavior for active runs and `run_not_active` terminal cases.

## End-to-End / Manual
- Anonymous user starts run from UI and receives live telemetry.
- Unsafe targets are blocked with readable API + UI error messaging.
- Dashboard transitions to terminal state after run stop/finish.
- Run data older than retention window is pruned without removing active runs.

## Launch Smoke
- 20 concurrent `POST /runs` attempts (validate stability and rate limiting).
- 100 short runs over 1 hour (check crash-free operation).
- Observe:
  - API p95 latency
  - SSE continuity/reconnect behavior
  - memory stability (no unbounded growth)
  - retention pruning behavior

## Verification Commands
- Backend:
  - `cd backend && go test ./...`
  - `cd backend && go vet ./...`
- Frontend:
  - `cd frontend && bun run lint`
  - `cd frontend && bun run build`

