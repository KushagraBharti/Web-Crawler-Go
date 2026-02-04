# Implementation Plan

## Phase 0: Repo Skeleton
1. Create `/backend`, `/frontend`, `/infra` directories.
2. Add Docker Compose with Go API, Postgres, Next.js services.
3. Add basic README updates with run commands once commands exist.

## Phase 1: Steel Thread MVP
1. Backend API skeleton with run lifecycle endpoints.
2. In-memory scheduler with bounded queues.
3. Shared HTTP client and tuned transport.
4. Streaming HTML tokenization for link extraction.
5. In-memory dedup and minimal canonicalization.
6. Store run and page metadata in Postgres.
7. Telemetry aggregator for SSE frames.
8. Frontend dashboard with live charts and graph.

## Phase 2: Reliability and Metrics
1. Redirect rescheduling.
2. Strict timeouts and size caps.
3. Retry policy with backoff and jitter.
4. Circuit breaker per host.
5. httptrace metrics for connection reuse.
6. pprof endpoints.

## Phase 3: Robots and Politeness
1. Robots cache and allow/disallow checks.
2. Herd-proof robots fetching.
3. Robots state visible in UI.

## Phase 4: UI Polish
1. Per-host table with sorting and filtering.
2. Domain personality cards.
3. Smooth canvas rendering for domain graph.
4. Optional live tuning controls.

## Deliverable Checklist
- One command starts backend, frontend, and DB.
- Crawl starts from UI and streams live updates.
- Crawler stops correctly at limits or on stop.
- Metrics and pprof are available.
- Tests pass and failure modes are documented.