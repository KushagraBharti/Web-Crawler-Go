# API

## Base
- REST: JSON over HTTP.
- Realtime: Server-Sent Events (SSE).

## Error Schema
All API errors follow:
```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid run configuration",
    "details": {
      "max_pages": "must be <= 2000"
    }
  }
}
```

Common codes:
- `validation_error`
- `invalid_scheme`
- `blocked_hostname`
- `blocked_ip`
- `host_resolution_failed`
- `rate_limited`
- `run_not_found`
- `run_not_active`
- `start_run_failed`
- `stop_run_failed`

## Launch Caps (Server Enforced)
- `max_pages <= 2000`
- `time_budget_seconds <= 300`
- `global_concurrency <= 32`
- `per_host_concurrency <= 4`
- `max_links_per_page <= 100`

## Endpoints

### POST /runs
Create a crawl run.

Request
```json
{
  "seed_url": "https://example.com",
  "max_depth": 3,
  "max_pages": 1000,
  "time_budget_seconds": 300,
  "max_links_per_page": 100,
  "global_concurrency": 16,
  "per_host_concurrency": 3,
  "user_agent": "Crawler/1.0",
  "respect_robots": true
}
```

Notes:
- Seed URL must be HTTP/HTTPS.
- Localhost/internal/private/reserved targets are blocked.
- Per-IP rate limit is applied.

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
  "stop_reason": "running",
  "limits": {
    "max_depth": 3,
    "max_pages": 1000,
    "time_budget_seconds": 300
  },
  "summary": {
    "pages_fetched": 1200,
    "pages_failed": 40,
    "unique_hosts": 180,
    "total_bytes": 9823456,
    "last_fetched_at": "timestamp"
  },
  "stats": {
    "pages_fetched": 1200
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

If the run is not active, returns:
- `409 run_not_active` (with terminal status in `error.details`), or
- `404 run_not_found`.

### GET /runs/{id}/pages
List most recent pages collected for a run.

Query:
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

### GET /healthz
Liveness endpoint.

Response
```json
{
  "status": "ok",
  "time": "timestamp"
}
```

### GET /readyz
Readiness endpoint.

Response
```json
{
  "status": "ready",
  "time": "timestamp"
}
```

If not ready: `503` with structured error.

### GET /metrics
Prometheus-style metrics.

### GET /debug/pprof/
Go pprof endpoints.

