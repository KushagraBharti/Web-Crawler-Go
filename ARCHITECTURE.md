# Architecture

## Core Flow
1. User creates a run with `mode = url | keyword`.
2. Backend resolves the seed:
   - URL mode canonicalizes the input URL.
   - Keyword mode fetches DuckDuckGo HTML results and selects the top result as the primary seed.
3. The crawler starts from that seed URL.
4. Each fetched page is converted into:
   - title
   - readable text
   - excerpt
   - outgoing links
5. Accepted discovered pages become tree nodes with parent-child edges.
6. The run manager writes `run.json`, `pages.json`, `tree.json`, and `diagnostics.json`.
7. The frontend renders:
   - run summary
   - page list
   - page detail view
   - crawl tree
   - diagnostics paths

## Components
- API server: run lifecycle, run snapshots, page detail, tree, diagnostics, SSE
- Run manager: owns in-memory run state, SSE subscribers, and artifact persistence
- Search resolver: keyword -> DuckDuckGo HTML -> primary seed URL
- Scheduler: bounded frontier with per-host fairness and robots gating
- Engine: fetch workers, retries, redirects, and content extraction
- Extractor: title, readable text, excerpt, outgoing links
- Artifact store: file-backed JSON persistence under `data/runs/<run-id>/`
- Frontend: result-first UI for browsing pages and tree state

## Data Model
- Run: config, seed resolution, timestamps, status, summary, artifact paths
- Page: stable page ID, parent page ID, URL, canonical URL, title, text, excerpt, status, timings, outgoing links
- Tree: nodes and parent-child edges keyed by page ID
- Diagnostics: skipped URLs, retries, errors, fetch log, seed search results

## Deliberate Simplifications
- Postgres is optional and not required for v1.
- The graph is page-based, not host-based.
- Diagnostics exist, but they are secondary to page results.
- Content extraction is readable-text oriented, not raw HTML storage.
