# Architecture

## Core Flow

1. User creates a run with `mode = url | keyword`.
2. Backend resolves the seed:
   - **URL mode**: canonicalizes the input URL directly.
   - **Keyword mode**: calls the **Brave Search API** and selects the top result as the primary seed.
3. The crawler starts from the seed URL (depth 0). That page is the tree root.
4. Each fetched page produces:
   - title, readable text, excerpt
   - outgoing links
5. Accepted discovered pages become tree nodes with parent→child edges.
   - Edges reflect discovery order: "page B was found in page A's links."
   - Pages discovered from the same parent are siblings — no edge between them.
6. The run manager writes `run.json`, `pages.json`, `tree.json`, and `diagnostics.json` under `data/runs/<run-id>/`.
7. The frontend renders:
   - a page index (left column)
   - a reading surface for selected page content (right, "Read" tab)
   - the rooted discovery tree (right, "Graph" tab)
   - a collapsible diagnostics footer

## Components

- **API server**: run lifecycle (create / start / stop / delete), snapshots, page detail, tree, diagnostics, SSE
- **Run manager**: in-memory run state, SSE subscribers, artifact persistence
- **Search resolver**: keyword → Brave Search API → primary seed URL
- **Scheduler**: bounded frontier with per-host fairness and robots gating
- **Engine**: fetch workers, retries, redirects, body limits, dedup, content extraction
- **Extractor**: title, readable text (Readability-style), excerpt, outgoing links
- **Artifact store**: file-backed JSON persistence under `data/runs/<run-id>/`
- **Frontend**: editorial dark UI — page index, article reading surface, rooted graph, diagnostics

## Data Model

- **Run**: config, seed resolution, timestamps, status, summary, `root_page_id`
- **Page**: stable page ID, parent page ID, URL, canonical URL, title, text, excerpt, status, timings, outgoing links
- **Tree**: nodes (id, depth, title, url) + parent→child edges keyed by page ID; root is the depth-0 seed page
- **Diagnostics**: skipped URLs, retries, errors, fetch log, seed search results
- **Snapshot**: the full run state serialised in one JSON blob; `root_page_id` identifies the seed node for deterministic tree rendering

## Rooted Discovery Tree

The crawl tree is a **rooted directed tree**, not a general web graph:
- The seed page is always the root (depth 0).
- An edge A→B means: B was discovered from a link found on A.
- Two pages discovered from the same parent share no edge between them.
- The backend exposes `root_page_id` on every Snapshot so the frontend never has to guess.

## Deliberate Simplifications

- Postgres is optional and not wired for v1; local JSON artifacts are the persistence layer.
- The graph is page-based and tree-shaped, not a general link graph.
- Diagnostics exist but are secondary; they live in a collapsed footer panel.
- Content extraction targets readable text, not raw HTML.
