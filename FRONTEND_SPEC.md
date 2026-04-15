# Frontend Spec

## User Flow

1. **Home (`/`)**: masthead + big editorial headline + launch form (right column).
2. User enters a URL or keyword, optionally adjusts depth / max pages / time budget.
   Advanced controls (concurrency, robots) are hidden under a collapsible section.
3. Submit → `POST /runs` → `POST /runs/:id/start` → redirect to `/runs/:id`.
4. **Run workspace (`/runs/:id`)**:
   - Server-side renders initial snapshot (SSR fetch from backend).
   - Client subscribes to SSE stream at `GET /runs/:id/events` for live updates.
   - Periodic poll every 12 seconds as a fallback for dropped SSE frames.

## Workspace Layout

```
┌─ STATUS BAR ──────────────────────────────────────────────┐
│  ← Arachne    [title]              [stats] [live] [stop]  │
├─────────────────────────────────────────────────────────── │
│  INDEX (288px)  │  Read ─── Graph                         │
│  ─────────────  │  ────────────────────────────────────── │
│  01 Title       │  SEED                                   │
│  02 Title       │  Article title (Fraunces serif)         │
│  03 Title       │  url · status · depth · latency         │
│  …              │  ──────────────────────────────────────  │
│                 │  Body text in Newsreader serif           │
│                 │  (white-space: pre-wrap, max 66ch)       │
├─────────────────────────────────────────────────────────── │
│  ▶ DIAGNOSTICS (collapsed)                                 │
└─────────────────────────────────────────────────────────── │
```

## Components

| Component | Role |
|-----------|------|
| `RunForm` | Mode toggle (URL/Keyword), input, basic config, advanced section |
| `RunWorkspace` | Orchestrates state: SSE, poll, tab, page selection, stop |
| `PageSidebar` | Numbered page index; red flash on new arrivals |
| `PageDetail` | Hero reading surface: skeleton loading, article body in Newsreader |
| `TreeView` | SVG rooted tree; root = `snapshot.root_page_id`; edges parent→child |
| `DiagnosticsPanel` | Collapsed `<details>` footer with stats, seed, artifact paths |

## Live Updates

- SSE frames carry `new_pages`, `tree_nodes`, `tree_edges` incrementally.
- On `status === 'stopped'`: one-shot full refetch of run + tree + diagnostics.
- Fallback poll every 12 seconds while `streamState !== 'stopped'`.
- All async errors surface as visible inline messages (no silent `.catch`).

## Tree Semantics

- Root node = `snapshot.root_page_id` (depth-0 seed page; backend always sets this).
- Edges are parent→child discovery edges only.
- Nodes and edges may arrive in separate SSE frames; orphaned nodes placed at bottom.
- Clicking a tree node selects it and loads the page in the Read tab.

## Aesthetic

- **Background**: `#0c0b09` warm near-black.
- **Display type**: Fraunces (700/900) for headlines, run titles, article titles.
- **Reading type**: Newsreader for article body text (1.1rem, line-height 1.88, max 66ch).
- **UI chrome**: Barlow Condensed for all labels, stats, tabs, buttons.
- **Accent**: `#c8382a` editorial red — section labels, active tab underline, active index marker.
- **Status**: `#4cba86` green with pulse animation for live state.
- **Corners**: sharp (border-radius 2–4px).
- **Rules**: thin `rgba(240,235,226,0.08–0.14)` horizontal lines as structural dividers.

## Keyword Mode Note

Requires `BRAVE_SEARCH_API_KEY` to be set in the backend environment.
