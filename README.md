# Arachne

Result-first web crawler in Go with a Next.js UI. Start from a URL or keyword, crawl outward, read the extracted page content, inspect the crawl tree, and verify everything through local JSON artifacts.

High Level System Design

## Repo Structure

- `backend/` Go API, crawler runtime, search resolver, JSON artifact writer
- `frontend/` Next.js UI for starting crawls and inspecting results
- `infra/` Docker Compose
- `data/runs/` local JSON artifacts written per crawl run

## What Changed

- The old telemetry-first dashboard has been replaced by a results-first workflow.
- The graph is now a rooted page-discovery tree, not a host network graph.
- The crawler stores readable page content, excerpts, tree edges, and diagnostics in JSON files under `data/runs/<run-id>/`.

## Quickstart

### Backend

```bash
cd backend
# copy .env.example to .env and set your Brave key once
go run ./cmd/server
```

PowerShell:

```powershell
cd backend
Copy-Item .env.example .env
# edit .env and set BRAVE_SEARCH_API_KEY once
go run ./cmd/server
```

Get the key from Brave's API dashboard:

- Create an account: [Brave Search API Quickstart](https://api-dashboard.search.brave.com/documentation/quickstart)
- In the dashboard, subscribe to a plan, then go to `API Keys` and create a key.

### Frontend

```bash
cd frontend
bun install
bun run dev
```

Then open:

- UI: [http://localhost:3000](http://localhost:3000)
- API: [http://localhost:8080](http://localhost:8080)

## Docker

```bash
docker compose -f infra/docker-compose.yml up --build
```

## Terminal-Friendly Verification

Create a run:

```bash
curl -X POST http://localhost:8080/runs \
  -H "Content-Type: application/json" \
  -d '{"mode":"keyword","input":"Alan Turing","max_depth":2,"max_pages":20}'
```

Start it:

```bash
curl -X POST http://localhost:8080/runs/<run-id>/start
```

Inspect artifacts:

```bash
ls data/runs/<run-id>
cat data/runs/<run-id>/run.json
cat data/runs/<run-id>/tree.json
cat data/runs/<run-id>/pages.json
cat data/runs/<run-id>/diagnostics.json
```

## Key Features

- URL and keyword entry modes
- Brave Search API seed resolution for keyword mode
- Bounded crawler with dedup, retries, robots handling, timeouts, and per-host fairness
- Readable content extraction with title, body text, excerpt, and outgoing links
- Rooted page-discovery tree
- JSON artifact persistence for every run
- Result-first UI for browsing pages and diagnostics

## Verification

- Backend: `go test ./...`
- Frontend: `bun run build`

## Notes

- Postgres is no longer required for v1. Local JSON artifacts are the primary persistence layer.
- Keyword mode requires `BRAVE_SEARCH_API_KEY`.
- The backend auto-loads `backend/.env` if present.
- Update `README.md` whenever top-level directories or developer commands change.

