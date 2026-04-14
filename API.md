# API

## Base
- REST: JSON over HTTP
- Realtime: SSE over `/runs/{id}/events`

## Endpoints

### `GET /runs`
List recent runs with their latest snapshot.

### `POST /runs`
Create a crawl run and resolve its seed.

Request:
```json
{
  "mode": "keyword",
  "input": "Alan Turing",
  "max_depth": 2,
  "max_pages": 20,
  "time_budget_seconds": 180,
  "max_links_per_page": 25,
  "global_concurrency": 16,
  "per_host_concurrency": 4,
  "respect_robots": true
}
```

Response:
```json
{
  "id": "run-id",
  "status": "created",
  "created_at": "timestamp",
  "seed": {
    "query": "Alan Turing",
    "primary_url": "https://en.wikipedia.org/wiki/Alan_Turing",
    "results": ["https://en.wikipedia.org/wiki/Alan_Turing"]
  },
  "config": {}
}
```

### `POST /runs/{id}/start`
Start the crawl.

### `POST /runs/{id}/stop`
Stop the crawl.

### `DELETE /runs/{id}`
Delete a stopped run and remove its local JSON artifact directory.

### `GET /runs/{id}`
Return the full run snapshot.

Includes:
- run config
- seed resolution
- status and stop reason
- summary
- pages
- tree nodes and edges
- diagnostics
- artifact file paths

### `GET /runs/{id}/pages`
Return the crawled pages in discovery order.

### `GET /runs/{id}/pages/{pageId}`
Return the full extracted content for a single page.

### `GET /runs/{id}/tree`
Return:
```json
{
  "nodes": [],
  "edges": []
}
```

### `GET /runs/{id}/diagnostics`
Return:
- search seed data
- skipped URLs
- retry events
- error events
- fetch log
- artifact paths

### `GET /runs/{id}/events`
SSE stream of run progress.

Event: `frame`
```json
{
  "ts": "timestamp",
  "status": "running",
  "summary": {
    "status": "running",
    "pages_fetched": 3,
    "pages_failed": 0,
    "pages_queued": 4
  },
  "new_pages": [],
  "tree_nodes": [],
  "tree_edges": []
}
```

### `GET /health`
Simple health endpoint.
