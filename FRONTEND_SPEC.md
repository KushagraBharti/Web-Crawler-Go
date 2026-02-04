# Frontend Spec

## Stack
- Next.js + TypeScript + React.
- State: Zustand (lightweight) unless Redux is required later.
- Charts: lightweight chart lib or custom canvas.
- Realtime: SSE via `EventSource`.

## Pages
- `/` landing page with run creation form.
- `/runs/[id]` live dashboard.

## Run Creation Form
Fields
- Seed URL
- Max depth
- Max pages
- Time budget (seconds)
- Max links per page
- Global concurrency
- Per-host concurrency
- Respect robots (toggle)

Behavior
- On submit: `POST /runs`, then `POST /runs/{id}/start`, then navigate to dashboard.

## Dashboard Layout
Panels
- Throughput chart (pages/sec).
- Queue depths chart (frontier, fetch, parse).
- Error summary panel (top classes).
- Per-host table (inflight, p95, error rate).
- Domain graph (nodes = hosts, edges = discovered links).

SSE Updates
- Subscribe to `/runs/{id}/events`.
- Apply updates at 5 to 10 Hz.
- Use aggregated frames; avoid per-request updates.

## UX Notes
- Keep the UI responsive under high event rates.
- Show stop button with confirmation.
- Visualize backpressure via queue depth trends.