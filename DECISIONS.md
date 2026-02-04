# Decisions

This file captures defaults made during long-run autonomous work.

## Defaults
- Backend port: 8080
- Frontend port: 3000
- Postgres port: 5432
- Service names: backend, frontend, db
- API base path: /
- SSE frame rate: 5 to 10 Hz
- State management: Zustand
- Charts: lightweight library or custom canvas
- Redirect depth: redirects re-enqueue at the same depth (do not increase depth)
- Queue sizing: frontier = global concurrency * 200, fetch/parse = global concurrency * 4
- Max body bytes default: 1 MiB
- Personality cards: derived from host error rate, p95 latency, and inflight (client-side)

## Open to Change
- Choice of chart library.
- Exact UI layout and styling.
- Final schema tuning once data volume is clear.