# Decisions

This file captures defaults and tradeoffs used for launch hardening.

## Launch Defaults
- Backend port: `8080`
- Frontend port: `3000`
- Postgres port: `5432`
- Deployment model: Vercel frontend + Railway backend (hybrid)
- Storage retention: `72h` (short retention, no user accounts)
- Rate limit: `RUN_CREATE_RATE_LIMIT=10` per `RUN_CREATE_RATE_WINDOW=1m`

## Server-Enforced Run Caps
- `max_pages <= 2000`
- `time_budget_seconds <= 300`
- `global_concurrency <= 32`
- `per_host_concurrency <= 4`
- `max_links_per_page <= 100`

## Security/Abuse Guardrails
- Block non-HTTP(S) targets.
- Block localhost/internal hostnames.
- Block private/reserved IP targets (direct and DNS-resolved).
- Restrict CORS to explicit allowed origins (+ optional preview suffix).

## Reliability Defaults
- SSE frame rate: 5 Hz (`200ms` emit interval).
- Queue sizing: `frontier = globalConcurrency * 200`, `fetch/parse = globalConcurrency * 4`.
- Max response body: `1 MiB`.
- Retention sweep interval: `1h`.

## UX/Frontend Defaults
- Run form includes `safe` and `balanced` presets.
- Run-scoped store reset on run-id change.
- Structured API errors surfaced in UI.
- Stop action exposes explicit success/failure feedback.

