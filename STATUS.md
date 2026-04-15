# Status

## Current State

- v2 redesign complete
- Backend exposes `root_page_id` on every Snapshot for deterministic tree root detection
- Frontend rebuilt with editorial dark aesthetic (Fraunces / Newsreader / Barlow Condensed)
- Reading surface (article body) is the hero; tree and page index are supporting panels
- "Read" / "Graph" tab structure keeps the workspace focused
- Tree root is always the seed page — never inferred from heuristics
- SSE + periodic 12-second poll as fallback for dropped frames; no silent error swallowing
- Diagnostics collapsed by default
- Keyword mode resolves seeds through the **Brave Search API** (`BRAVE_SEARCH_API_KEY` required)

## Verification

- `go build ./...` passes in `backend/`
- `go test ./...` passes in `backend/`
- `bun run build` passes in `frontend/`

## Notes

- `data/runs/` is gitignored and intended for local inspection
- Postgres and `internal/storage` exist but are not wired in v1 — JSON artifacts only
- `internal/metrics` and `pprof` import are present but unused — clean-up tracked separately
