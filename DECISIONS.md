# Decisions

## Current Defaults
- Primary persistence: local JSON files in `data/runs/`
- Keyword seed source: DuckDuckGo HTML
- Graph model: rooted page-discovery tree
- Primary run summary surface: page results, not host telemetry
- Default crawl depth: 2
- Default page cap: 50
- Default time budget: 3 minutes
- Default global concurrency: 16
- Default per-host concurrency: 4
- Default max links per page: 25
- Default max body size: 2 MiB

## Deliberate Product Choices
- Break backward compatibility to simplify the product
- Keep concurrency, retries, robots, and politeness behavior
- Store readable text and excerpts rather than raw HTML by default
- Keep diagnostics available through JSON artifacts and API endpoints, but not as the main UI

## Open To Change
- Search provider abstraction
- Artifact retention and cleanup policy
- Optional database mirror
- Richer tree layouts and filtering
