# Arachne

Arachne is a result-first web crawler built with a Go backend and a Next.js
frontend. It is designed for the workflow people actually want from a crawler:
search or enter a URL, pick the best starting page, read the extracted content,
and inspect exactly how the crawl expanded through the web.

The system is intentionally not a telemetry dashboard first. The primary output
is readable page content, a rooted discovery tree, diagnostics, and local JSON
artifacts that can be inspected without the browser.

## What It Does

Arachne supports two entry modes:

- **URL mode**: start directly from a URL.
- **Keyword mode**: search through the Brave Search API, show the top 10 results,
  let the user pick the seed, and prefetch those candidate pages in the
  background so the selected root feels instant.

Once a run starts, the crawler fetches the root page, extracts readable text and
links, then crawls outward from accepted outgoing links until it hits configured
limits such as max depth, max pages, time budget, robots rules, or deduplication.

## Current User Flow

1. Enter a URL or keyword.
2. For keyword mode, the backend calls Brave Search and returns the top 10
   results immediately.
3. While the user is choosing a result, the backend concurrently prefetches
   those top 10 pages.
4. Click `Start here` on a result.
5. If that result has already prefetched, the page is emitted immediately as the
   depth-0 root.
6. The crawler then continues from that root page's outgoing links.
7. The UI streams new pages live, renders the readable article text, and shows
   the discovery tree.

Example: if you search `Alan Turing` and choose the PBS result, PBS becomes the
tree root. Wikipedia may still appear later if the PBS page links to it, but it
is no longer the starting page.

## Why It Feels Fast

Keyword search and candidate-page fetching are split:

- Brave result resolution is blocking because the UI needs the top 10 choices.
- Candidate page prefetch starts immediately after search results are known.
- The user can read and choose while the backend is already fetching those pages.
- Starting from a prefetched result avoids doing the first page fetch after the
  user clicks `Start here`.
- If the chosen page is not ready yet, the backend waits briefly for that one
  prefetch and then falls back to a normal fetch so the run does not hang.

This gives the interaction a search-engine feel: the user sees choices quickly,
and the selected root page is usually ready by the time they commit.

## Architecture

```text
frontend/
  Next.js App Router UI
  run form, seed selection, page reader, tree view, diagnostics

backend/
  Go HTTP API
  run manager
  Brave Search resolver
  crawler engine
  scheduler, robots handling, dedup, retries
  JSON artifact store

data/runs/
  local run artifacts written per crawl

infra/
  Docker Compose support
```

Core backend flow:

1. `POST /runs` creates a run.
2. URL mode canonicalizes the input URL.
3. Keyword mode calls Brave Search, keeps the top 10 URLs, and starts background
   prefetch for those pages.
4. `POST /runs/{id}/start` starts the crawler from the selected `seed_url`.
5. The engine emits the root page, discovers links, builds tree nodes and edges,
   and streams updates through SSE.
6. The run manager persists `run.json`, `pages.json`, `tree.json`, and
   `diagnostics.json`.

## Crawler Model

The graph is a rooted page-discovery tree, not a general web graph.

- The seed page is always depth `0`.
- A tree edge `A -> B` means page `B` was discovered from a link on page `A`.
- Pages discovered from the same parent are siblings.
- Duplicate canonical URLs are skipped.
- The backend exposes `root_page_id` so the frontend never guesses which page is
  the root.

The crawler is bounded and polite:

- global concurrency
- per-host concurrency
- request/header/TLS/idle timeouts
- response body size limit
- retry handling
- redirect handling
- robots.txt support
- max depth
- max pages
- time budget
- max links per page

## Performance

Performance depends heavily on the public internet: remote server latency,
robots.txt, redirects, TLS negotiation, rate limits, page size, and how many
links survive canonicalization. For a stable baseline, this repo includes a
local synthetic benchmark that removes external network noise.

Benchmark command:

```powershell
cd backend
go test ./internal/crawler -run '^$' -bench BenchmarkSyntheticCrawl501Pages -benchtime=5x -count=1
```

Result from this machine:

```text
CPU: 13th Gen Intel(R) Core(TM) i7-13700HX
BenchmarkSyntheticCrawl501Pages-24    5    1508935180 ns/op
```

Interpretation:

- Workload: 1 local root page plus 500 linked local pages.
- Total pages per benchmark iteration: 501.
- Mean time per iteration: about 1.51 seconds.
- Approximate throughput: about 332 pages/sec.
- Environment: local `httptest` HTTP server, no external network, 64 global
  concurrency, 64 per-host concurrency, 1 MB body limit.

This is not a promise for public websites. It is a controlled measurement of
the crawler's local scheduling, fetch, extraction, dedup, and tree-emission path.
Real crawls will usually be slower because public sites add network and policy
costs. The synthetic benchmark is useful because it is repeatable.

## Quickstart

Prerequisites:

- Go 1.22+
- Bun
- Brave Search API key for keyword mode

Backend:

```powershell
cd backend
go run ./cmd/server
```

Optional `backend/.env` for keyword mode:

```text
BRAVE_SEARCH_API_KEY=your_key_here
```

The backend auto-loads `backend/.env` when present. URL mode works without a
Brave key. Keyword mode requires `BRAVE_SEARCH_API_KEY`.

Frontend:

```powershell
cd frontend
bun install
bun run dev
```

Open:

- UI: http://localhost:3000
- API: http://localhost:8080

## API Examples

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

Inspect the run:

```powershell
curl http://localhost:8080/runs/<run-id>
curl http://localhost:8080/runs/<run-id>/pages
curl http://localhost:8080/runs/<run-id>/tree
curl http://localhost:8080/runs/<run-id>/diagnostics
```

## Artifacts

Each run writes local JSON files under:

```text
data/runs/<run-id>/
```

Files:

- `run.json`: run config, selected seed, status, summary, artifact paths
- `pages.json`: extracted page records in discovery order
- `tree.json`: tree nodes and parent-child edges
- `diagnostics.json`: search attempts, selected seed, prefetch results, skips,
  retries, errors, fetch log

These files are intentionally easy to inspect. They are the ground truth for
debugging whether the selected seed was honored, what was fetched, and why URLs
were skipped.

## Verification

Backend:

```powershell
cd backend
go test ./...
```

Frontend:

```powershell
cd frontend
bun run build
```

Benchmark:

```powershell
cd backend
go test ./internal/crawler -run '^$' -bench BenchmarkSyntheticCrawl501Pages -benchtime=5x -count=1
```

Current verification status:

- `go test ./...` passes.
- `bun run build` passes.
- Synthetic crawler benchmark completes successfully.

## Docker

```powershell
docker compose -f infra/docker-compose.yml up --build
```

Note: Postgres is present in the compose file and older storage scaffolding still
exists, but v1 persistence currently uses local JSON artifacts. The app does not
require Postgres for normal local use.

## Design Notes

- The frontend uses an editorial dark interface focused on reading extracted
  content.
- The `Read` tab is the primary surface.
- The `Graph` tab shows the rooted discovery tree.
- Diagnostics stay available but collapsed by default.
- SSE streams crawl updates live, with periodic polling as a fallback.

## Limitations

- The crawler does not render JavaScript. It fetches HTML and extracts text and
  links from the response body.
- Search quality depends on Brave Search API results.
- Public-web crawl speed depends on remote websites, rate limits, robots.txt,
  redirects, and network latency.
- Postgres-related code is present but not wired into the current v1 runtime.
- Raw HTML archival is not the default output.

