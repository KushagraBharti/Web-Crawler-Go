# Frontend Spec

## Stack
- Next.js App Router
- TypeScript + React
- No client-side charting requirement for v1

## Pages
- `/`
  - URL or keyword run creation
  - advanced crawl controls hidden behind disclosure
- `/runs/[id]`
  - run summary and diagnostics
  - page list
  - page detail reader
  - rooted crawl tree

## Primary UX
- User enters a URL or keyword.
- UI starts the run and redirects into the workspace.
- Workspace streams new pages over SSE.
- User clicks any page to read extracted content.
- User can visually verify the crawl tree and copy artifact file paths for terminal inspection.

## Visual Direction
- Research workbench, editorial typography, warm paper palette
- Calm and legible instead of dashboard-heavy
- Tree view and page reading are the two primary surfaces

## Non-Goals for v1
- Dense telemetry panels
- Host heatmaps
- Personality scoring
- Fancy graph physics
