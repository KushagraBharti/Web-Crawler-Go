# Project Spec

## Summary
Build a result-first web crawler in Go with a simple frontend. A user should be able to enter a URL or keyword, crawl outward from the resolved seed, read the extracted content of discovered pages, inspect the crawl tree, and verify crawl output through local JSON artifacts.

## Goals
- Make the first successful crawl easy to understand
- Keep the crawler bounded, concurrent, polite, and reliable
- Persist enough output to verify correctness from both the UI and terminal
- Represent crawl structure as a rooted page-discovery tree

## Non-Goals
- Telemetry-heavy dashboard as the primary experience
- Distributed crawling
- Browser rendering
- Raw HTML archival by default

## Success Criteria
- URL mode and keyword mode both produce a usable crawl
- A user can click a page and read extracted content
- The tree clearly shows parent-child discovery flow
- JSON artifacts make debugging possible without the browser
- Backend tests and frontend build pass
