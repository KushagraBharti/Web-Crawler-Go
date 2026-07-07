# Arachne

Arachne is a from-scratch Go web crawler with a live reading UI. It is built to show systems judgment: host-fair scheduling, bounded concurrency, robots.txt gating, circuit breakers, retries, PostgreSQL persistence, and real public-web benchmark evidence rather than a wrapper around an existing crawler framework.

The crawler has two standout benchmark stories:

- `10,003` unique HTML pages in `15.39s`, about `650 pages/sec`, in a fetch-only public-web burst benchmark at `2,000` global concurrency and `16` per-host concurrency.
- A separate `100,000` successful-page public-web run in about `31` minutes, with `283k` attempted fetches, `1.25M` discovered unique URLs, `11.1M` duplicate suppressions, and `18.4 GB` downloaded.

The frontend is intentionally a reading surface, not just telemetry. Pages stream in over SSE, the selected page renders extracted article text, and the graph view shows the discovery tree. Keyword mode uses the Brave Search API to find candidate seeds and prefetches the top results so the selected root usually opens immediately.

It started as this sketch:

![Original system sketch](Web-Crawler-Go_Basic_System_Ideation.jpeg)

## Technical Architecture

The crawler core lives in `backend/internal/crawler` and uses plain goroutines, channels, and hand-built scheduling primitives. There is no crawl framework and no queue library underneath.

### Scheduler

Scheduler truth is round-robin host queues:

- Incoming URLs are canonicalized and partitioned into per-host FIFO queues.
- A single scheduling loop ticks every `5 ms`.
- Hosts are walked round-robin so a link-heavy domain cannot starve the frontier.
- Fetch workers are a goroutine pool sized to `max(4, global concurrency)`.

Every dispatch passes the same gate sequence:

1. Frontier capacity.
2. Not-before timestamp for retries and `429` holds.
3. Per-host circuit breaker.
4. Global semaphore.
5. Per-host semaphore.
6. robots.txt allow/deny readiness.

A dispatched task carries both global and per-host permits. If the fetch pool cannot accept the task, permits are released and the task goes back to the front of its host queue with a short hold, creating backpressure without piling work into an unbounded buffer.

### Politeness And Failure Handling

- Dual semaphores enforce global and per-host concurrency.
- robots.txt is fetched per host and cached for 24 hours.
- While a host's robots file is still in flight, tasks defer instead of guessing.
- Per-host circuit breakers open after five consecutive errors for 30 seconds, then use a half-open probe.
- `429` responses honor `Retry-After`.
- Other retries use exponential backoff from a 300 ms base, with two retries by default.
- Response bodies cap at 2 MB.
- Request, header, and TLS handshake timeouts default to 15 s, 10 s, and 8 s.

### Persistence And Artifacts

Runs persist to PostgreSQL through `backend/internal/storage`: runs, pages, host state, per-host stats, host-to-host link edges, and crawl errors. Go migrations create the schema, and retention pruning removes old runs. `infra/docker-compose.yml` provisions Postgres 16 and wires `DATABASE_URL` into the backend.

Each run also writes portable JSON artifacts under `data/runs/<run-id>/`:

- `run.json`: config, selected seed, status, summary
- `pages.json`: extracted page records in discovery order
- `tree.json`: tree nodes and parent-child edges
- `diagnostics.json`: search attempts, prefetch results, skips, retries, errors, fetch log

The JSON files are the UI's reading source and a useful audit trail when debugging why a URL was skipped.

### Traversal And UI

Traversal is BFS from a single selected seed. Canonical URL dedup ensures the same page under different spellings is fetched once. The emitted graph is a rooted discovery tree: edge `A -> B` means B was first found on A.

The Next.js frontend subscribes to SSE at `GET /runs/:id/events`, with a 12-second poll fallback for dropped frames. New pages flash into the sidebar index, the Read tab shows extracted article text, the Graph tab renders the discovery tree as SVG, and diagnostics stay collapsed until needed.

Keyword mode resolves a phrase through Brave Search into ten candidate seeds. The backend prefetches all ten results in the background while the user chooses; if the selected page has already arrived, it becomes the depth-0 root immediately. URL mode skips search and starts directly from the submitted URL.

### Benchmarks

Public-web benchmark command:

```powershell
cd Web-Crawler-Go\backend
go run ./cmd/benchcrawl -target 10000 -ramp 500,1000,2000,3000 -per-host 16 -timeout 5s
```

Calibration on a 13th-gen i7 laptop:

| Global concurrency | Pages | Time | Pages/sec |
| --- | ---: | ---: | ---: |
| 500 | 10,001 | 30.45 s | 328 |
| 1,000 | 10,003 | 17.10 s | 585 |
| 2,000 | 10,003 | 15.39 s | 650 |
| 3,000 | 10,001 | 15.77 s | 634 |

The `100k` run at the `2,000` setting averaged `53 pages/sec` over `31.4` minutes. That long-run average is lower than the burst result because the public frontier accumulates slow hosts, TLS failures, rate limits, and robots constraints. Full JSON reports are committed under `benchmark-results/`.

Network-free synthetic benchmark:

```powershell
cd Web-Crawler-Go\backend
go test ./internal/crawler -run '^$' -bench BenchmarkSyntheticCrawl501Pages -benchtime=5x -count=1
```

That benchmark pushes 501 local pages through fetch, extract, dedup, and tree emission in about `1.51s` per iteration, roughly `332 pages/sec` end to end.

## Setup And Run

Prerequisites:

- Go `1.22+`
- Bun
- Docker, if using the compose path
- Brave Search API key for keyword mode; URL mode works without it

Run the backend:

```powershell
cd Web-Crawler-Go\backend
go run ./cmd/server
```

Optional `backend/.env`, auto-loaded when present:

```text
BRAVE_SEARCH_API_KEY=your_key_here
```

Run the frontend:

```powershell
cd Web-Crawler-Go\frontend
bun install
bun run dev
```

Open:

- UI: `http://localhost:3000`
- API: `http://localhost:8080`

Run the full stack with Docker:

```powershell
cd Web-Crawler-Go
docker compose -f infra/docker-compose.yml up --build
```

Create a keyword run:

```powershell
curl -X POST http://localhost:8080/runs `
  -H "Content-Type: application/json" `
  -d '{"mode":"keyword","input":"Alan Turing","max_depth":2,"max_pages":20}'
```

Start from a selected search result:

```powershell
curl -X POST http://localhost:8080/runs/<run-id>/start `
  -H "Content-Type: application/json" `
  -d '{"seed_url":"https://www.pbs.org/newshour/science/8-things-didnt-know-alan-turing"}'
```

Inspect a run:

```powershell
curl http://localhost:8080/runs/<run-id>
curl http://localhost:8080/runs/<run-id>/pages
curl http://localhost:8080/runs/<run-id>/tree
curl http://localhost:8080/runs/<run-id>/diagnostics
```

Validate:

```powershell
cd Web-Crawler-Go\backend
go test ./...
```

```powershell
cd Web-Crawler-Go\frontend
bun run build
```

## Repository Map

```text
Web-Crawler-Go/
├── backend/              # Go API, crawler engine, storage, benchmarks
├── frontend/             # Next.js reading UI
├── infra/                # Docker Compose with Postgres 16
├── benchmark-results/    # committed public-web benchmark reports
├── API.md                # endpoint contract
├── ARCHITECTURE.md       # system architecture notes
└── SCHEMA.md             # persistence schema notes
```

## Limitations

- No JavaScript rendering; the crawler fetches HTML and extracts text/links from the response body.
- No resume support; the live frontier is in memory.
- The `650 pages/sec` figure is a fetch benchmark, not artifact-writing product crawl throughput.
- Keyword mode depends on Brave Search API quality and availability.
- Public-web speed depends on remote servers, robots.txt, redirects, rate limits, and network latency.
