# Arachne — Go Web Crawler

A high-concurrency Go web crawler with breadth-first discovery, a host-partitioned frontier, robots-aware scheduling, PostgreSQL persistence, and a live reading interface.

Arachne crawled **10,003 public web pages in 67.96 seconds**, approximately **147.2 pages per second**, while preserving bounded concurrency, host fairness, URL deduplication, and inspectable crawl artifacts.

![Original system sketch](Web-Crawler-Go_Basic_System_Ideation.jpeg)

## Product and highlights

Arachne is both a crawler and a way to read the web it discovers.

Users can:

- Start from a direct URL or search phrase.
- Resolve keyword searches into candidate seeds through Brave Search.
- Watch new pages arrive live over server-sent events.
- Read extracted article content inside the application.
- Inspect the breadth-first discovery tree.
- Review retries, skips, robots decisions, and crawl errors.
- Return to persisted runs after the crawl has finished.

The crawler itself is built from standard Go concurrency primitives rather than an existing crawl framework. Its central design problem is not simply fetching pages quickly; it is distributing work fairly across hosts without losing control of backpressure, politeness, retries, or failure isolation.

## How Arachne works

### Crawl lifecycle

```text
URL or keyword
      ↓
seed resolution
      ↓
URL canonicalization
      ↓
breadth-first frontier
      ↓
host-fair scheduler
      ↓
bounded fetch workers
      ↓
HTML extraction and link discovery
      ↓
PostgreSQL + JSON artifacts
      ↓
SSE reading interface
```

### Seed resolution

A direct URL becomes the depth-zero seed immediately.

In keyword mode, the backend queries Brave Search for candidate pages. The top results are prefetched while the user selects a root. If the selected page has already been fetched, it becomes the crawl root without another network round trip.

### Breadth-first frontier

Discovered URLs are canonicalized before entering the frontier. Canonicalization normalizes equivalent URL forms so the same page is not fetched repeatedly.

Traversal is breadth-first:

1. Fetch the current depth.
2. Extract links from successful HTML responses.
3. Add unseen links to the next frontier depth.
4. Record the first parent that discovered each page.
5. Continue until the page or depth limit is reached.

The resulting graph is a rooted discovery tree where `A → B` means page B was first discovered on page A.

### Host-fair scheduling

A single global FIFO queue would allow a link-heavy domain to dominate the crawl. Arachne instead partitions the frontier into per-host FIFO queues.

The scheduler:

- Maintains a rotating set of active hosts.
- Walks hosts round-robin.
- Preserves FIFO order within each host.
- Enforces both global and per-host concurrency.
- Requeues work when downstream workers are saturated.
- Applies retry and `429` not-before timestamps.

Every dispatch passes through:

1. Frontier-capacity checks
2. Retry and rate-limit timing
3. Per-host circuit-breaker state
4. Global concurrency semaphore
5. Per-host concurrency semaphore
6. `robots.txt` readiness and permission

This gives the crawler global throughput without allowing one host to consume every worker.

### Fetching, politeness, and failure handling

Fetch workers run as a bounded goroutine pool.

The crawler applies:

- Per-host `robots.txt` fetching and caching
- Deferred scheduling while robots policy is unresolved
- Global and per-host semaphores
- `Retry-After` handling for `429` responses
- Exponential retry backoff
- Per-host circuit breakers
- Request, header, and TLS timeouts
- Response-body size limits
- Content-type filtering
- Redirect and error diagnostics

A host circuit opens after repeated failures, waits before retrying, and then allows a half-open probe rather than repeatedly consuming the entire worker pool.

### Extraction and persistence

Successful HTML pages are parsed for:

- Canonical page metadata
- Title
- Readable text
- Outbound links
- Parent-child discovery relationships

Runs persist to PostgreSQL through the Go storage layer. Each run also writes portable JSON artifacts:

```text
run.json
pages.json
tree.json
diagnostics.json
```

These artifacts make the crawl inspectable even without the database and provide the frontend’s reading and graph surfaces.

### Live interface

The Next.js frontend subscribes to crawl events through SSE. A polling fallback recovers state if the live connection drops.

The interface exposes:

- A live page index
- Extracted article text
- Discovery depth
- SVG graph visualization
- Crawl progress
- Collapsible diagnostics

The interface is a live reading surface; the benchmark measures the crawler backend.

### Technologies and external dependencies

- **Crawler:** Go 1.22, goroutines, channels, semaphores
- **HTTP routing:** Chi
- **Database:** PostgreSQL through pgx
- **Robots support:** `robotstxt`
- **Metrics:** Prometheus client
- **Frontend:** Next.js, React, TypeScript, Zustand
- **Search:** Brave Search API
- **Infrastructure:** Docker Compose
- **Artifacts:** JSON and PostgreSQL

### Repository structure

```text
Web-Crawler-Go/
├── backend/
│   ├── cmd/server/             # HTTP API
│   ├── cmd/benchcrawl/         # Public-web benchmark runner
│   └── internal/
│       ├── crawler/            # Frontier, scheduler, fetching, extraction
│       └── storage/            # PostgreSQL persistence and retention
├── frontend/                   # Next.js live reading interface
├── infra/                      # Docker Compose and PostgreSQL
├── benchmark-results/          # Committed benchmark reports
├── data/runs/                  # Portable crawl artifacts
├── API.md                      # Endpoint contract
├── ARCHITECTURE.md             # System design
└── SCHEMA.md                   # Database schema
```

## Quick start

Requirements: Go 1.22+, Bun, Docker, and optionally a Brave Search API key.

Run the backend:

```powershell
cd backend
go run ./cmd/server
```

Run the frontend:

```powershell
cd frontend
bun install
bun run dev
```

Open:

- UI: `http://localhost:3000`
- API: `http://localhost:8080`

Run the full stack:

```powershell
docker compose -f infra/docker-compose.yml up --build
```

Validate:

```powershell
cd backend
go test ./...

cd ../frontend
bun run build
```
