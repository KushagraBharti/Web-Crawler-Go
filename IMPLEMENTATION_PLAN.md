# Implementation Plan

## Completed Rewrite
1. Replace telemetry-first API and dashboard with a result-first crawl workspace.
2. Add keyword seed resolution via DuckDuckGo HTML.
3. Simplify the crawler around page outputs:
   - fetch
   - extract readable content
   - enqueue discovered links
4. Persist run artifacts locally as JSON under `data/runs/<run-id>/`.
5. Expose run, page, tree, diagnostics, list, delete, and SSE endpoints.
6. Rebuild the frontend around:
   - run creation
   - page sidebar
   - page detail
   - rooted crawl tree
   - diagnostics panel
7. Rewrite docs to match the new product.

## Next Iteration Candidates
1. Better readable-content extraction heuristics.
2. Pagination or virtualized page lists for larger crawls.
3. Optional Postgres mirror for durable indexing.
4. Richer diagnostics management endpoints.
5. Better keyword result selection and fallback behavior.
