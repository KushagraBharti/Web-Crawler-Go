# Project Spec

## Summary
Build a high-performance web crawler in Go with a live dashboard. The crawler must be fast, bounded, and explainable under real-world web conditions (slow servers, redirects, traps, rate limits, flaky TLS). The dashboard must make throughput, backpressure, and failures legible in real time.

## Goals
- High throughput without instability or rudeness.
- Strict bounded memory via backpressure and fixed queues.
- Safe redirect handling (no politeness leaks).
- Connection reuse visibility and tuning.
- Live dashboard that reveals queue behavior, host-level throttling, and errors.

## Non-Goals (v1)
- Headless browser rendering.
- Distributed crawling.
- Bypassing bot protections.
- Full replay of historical runs in UI.

## Constraints
- HTTP and HTTPS only.
- Single-machine crawler.
- Live-only UI with aggregated updates.
- No raw HTML storage by default.

## MVP Scope
- Single reusable `http.Client` and tuned `http.Transport`.
- Bounded pipeline: frontier, fetch, parse queues.
- Per-host and global concurrency limits.
- Streaming HTML tokenization (no DOM) for link extraction.
- Minimal canonicalization and in-memory dedup.
- Basic limits: max depth, max pages, time budget, max links per page.
- SSE stream to UI with throughput, queue depths, top errors, and a simple domain graph.

## High-ROI Upgrades (Post-MVP)
- Redirect rescheduling through gates.
- Strict safety caps (timeouts, size limits, decompression caps).
- Retry policy with backoff and Retry-After for 429.
- Circuit breaker per host.
- Connection reuse metrics via httptrace.
- pprof endpoints for profiling.
- Robots.txt with herd-proofing.

## Success Criteria
- Crawls run for long periods without memory growth or deadlocks.
- UI identifies slow or failing hosts within 30 seconds.
- Connection reuse can be demonstrated and measured.
- Crawl stops cleanly at limits or on user stop.