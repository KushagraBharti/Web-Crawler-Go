# Test Plan

## Unit Tests
- Canonicalization and dedup behavior
- Readable content extraction
- Search result parsing from DuckDuckGo HTML
- Artifact writer creates expected JSON files
- Retry parsing and circuit-breaker transitions

## Integration Tests
- Local site with `A -> B -> D` and `A -> C`; verify:
  - all pages are fetched
  - `D` descends from `B`
  - `C` and `D` are not directly connected
- Keyword-mode seed resolution via mocked HTML search response
- Bounded queue behavior records skipped URLs in diagnostics

## Manual Verification
1. Start backend and frontend locally.
2. Launch a crawl from URL mode.
3. Launch a crawl from keyword mode.
4. Inspect:
   - UI page list and page detail
   - rooted tree
   - `data/runs/<run-id>/run.json`
   - `data/runs/<run-id>/pages.json`
   - `data/runs/<run-id>/tree.json`
   - `data/runs/<run-id>/diagnostics.json`
