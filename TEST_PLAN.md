# Test Plan

## Unit Tests
- URL canonicalization and normalization.
- Dedup set behavior under concurrency.
- Scheduler fairness and per-host limits.
- Robots allow/disallow matching.
- Retry and circuit breaker state transitions.

## Integration Tests
- Redirect handling re-enqueues through gates.
- Timeout handling and size caps with httptest servers.
- Content-type gating for parser.
- Queue backpressure under slow downstream.
- SSE stream formatting and pacing.

## End-to-End Tests
- Start a run, crawl a small local site, and stop.
- Verify limits: max pages, max depth, time budget.
- Verify no memory growth in steady-state small crawl.

## Performance Smoke
- Local load test with 1k pages in a controlled server.
- Check connection reuse rate and stable throughput.
- Record p50 and p95 latency and compare to baseline.