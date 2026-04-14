# Status

## Current State
- Result-first rewrite complete
- Backend writes local JSON artifacts for each run
- Frontend shows page explorer, page detail view, diagnostics, and rooted crawl tree
- Keyword mode resolves seeds through DuckDuckGo HTML

## Verification
- `go test ./...` passes in `backend/`
- `bun run build` passes in `frontend/`

## Notes
- `data/runs/` is gitignored and intended for local inspection
- Postgres is no longer required for the current product shape
