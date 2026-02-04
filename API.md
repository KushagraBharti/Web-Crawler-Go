# API

## Base
- REST: JSON over HTTP.
- Realtime: Server-Sent Events (SSE).

## Endpoints

### POST /runs
Create a crawl run.

Request
```json
{
  "seed_url": "https://example.com",
  "max_depth": 3,
  "max_pages": 5000,
  "time_budget_seconds": 600,
  "max_links_per_page": 200,
  "global_concurrency": 64,
  "per_host_concurrency": 4,
  "user_agent": "Crawler/1.0",
  "respect_robots": true
}
```

Response
```json
{
  "id": "uuid",
  "status": "created",
  "created_at": "timestamp"
}
```

### POST /runs/{id}/start
Start the crawl run.

Response
```json
{ "status": "running" }
```

### POST /runs/{id}/stop
Stop the crawl run.

Response
```json
{ "status": "stopped" }
```

### GET /runs/{id}
Fetch run status and summary stats.

Response
```json
{
  "id": "uuid",
  "status": "running",
  "created_at": "timestamp",
  "started_at": "timestamp",
  "stopped_at": null,
  "storage_mode": "memory",
  "stop_reason": "manual",
  "limits": {
    "max_depth": 3,
    "max_pages": 5000,
    "time_budget_seconds": 600
  },
  "summary": {
    "pages_fetched": 1200,
    "pages_failed": 40,
    "unique_hosts": 180,
    "total_bytes": 9823456,
    "last_fetched_at": "timestamp"
  },
  "stats": {
    "pages_fetched": 1200,
    "pages_per_sec": 25.4,
    "error_rate": 0.03
  }
}
```

### GET /runs/{id}/events
SSE stream of live dashboard frames.

Event: `frame`
```json
{
  "ts": "timestamp",
  "throughput": { "pages_per_sec": 25.4 },
  "queues": { "frontier": 1200, "fetch": 64, "parse": 32 },
  "errors": [ { "class": "timeout", "count": 12 } ],
  "hosts": [ { "host": "example.com", "inflight": 4, "p95_ms": 900 } ],
  "graph_delta": {
    "nodes": ["example.com"],
    "edges": [ ["example.com", "other.com", 3] ]
  }
}
```

### GET /runs/{id}/pages
List most recent pages collected for a run.

Query
```
?limit=50
```

Response
```json
{
  "items": [
    {
      "url": "https://example.com/page",
      "host": "example.com",
      "depth": 1,
      "status_code": 200,
      "content_type": "text/html",
      "fetch_ms": 120,
      "size_bytes": 34210,
      "error_class": "",
      "error_message": "",
      "fetched_at": "timestamp"
    }
  ]
}
```

### GET /metrics
Prometheus-style metrics.

### GET /debug/pprof/
Go pprof endpoints.
