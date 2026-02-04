# Architecture

## Components
- API server: run lifecycle, control, and SSE.
- Scheduler: fair selection across hosts with politeness limits.
- Frontier: bounded queue of canonicalized URLs.
- Fetcher: shared HTTP client, strict timeouts, size caps.
- Parser: streaming HTML tokenizer to extract links.
- Dedup: in-memory set for visited URLs (keyed by canonical URL).
- Storage: Postgres for runs, pages, host stats, and graph edges.
- Telemetry aggregator: aggregates high-frequency events into UI frames.

## Data Flow
1. Create run with seed URL and limits.
2. Canonicalize and dedup seed; enqueue into frontier.
3. Scheduler selects next URL based on host fairness and concurrency limits.
4. Fetcher downloads with strict limits and records metrics.
5. Parser extracts links and sends them back to the frontier.
6. Store per-page metadata, update host stats, and graph edges.
7. Telemetry aggregator emits SSE frames to the dashboard.

## Concurrency Model
- Bounded channels between stages.
- Global concurrency limit to cap total inflight requests.
- Per-host semaphore to avoid hammering a single host.
- Scheduler enforces fairness so hot hosts do not starve others.

## Redirect Handling
- Disable automatic redirects in the HTTP client.
- Treat `Location` as a newly discovered URL.
- Re-enqueue redirect targets through canonicalization, dedup, robots, and politeness gates.

## Failure Handling
- Classify errors (timeout, TLS, DNS, HTTP status, size limit, parse error).
- Retry only transient errors, with backoff and jitter.
- Circuit breaker per host to pause failing hosts.

## Observability
- Metrics: throughput, latency, queue depths, error taxonomy.
- httptrace for connection reuse, DNS, connect, and TLS timings.
- pprof for CPU and heap profiling.