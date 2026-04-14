# Schema

The current v1 product uses file-backed JSON artifacts instead of Postgres tables.

## `run.json`
- `run_id`
- `config`
- `seed`
- `status`
- `stop_reason`
- `created_at`
- `started_at`
- `stopped_at`
- `summary`
- `paths`

## `pages.json`
Array of page records:
- `id`
- `parent_page_id`
- `source_mode`
- `source_input`
- `url`
- `canonical_url`
- `host`
- `title`
- `text`
- `excerpt`
- `outgoing_links`
- `depth`
- `status_code`
- `content_type`
- `fetch_ms`
- `size_bytes`
- `error_class`
- `error_message`
- `discovered_at`
- `fetched_at`

## `tree.json`
- `nodes`
  - `id`
  - `parent_page_id`
  - `url`
  - `title`
  - `depth`
- `edges`
  - `parent_page_id`
  - `child_page_id`

## `diagnostics.json`
- `search_seed`
- `skipped_urls`
- `retry_events`
- `errors`
- `fetch_log`
- `artifact_dir`
- `artifact_files`
