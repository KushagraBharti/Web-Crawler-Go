'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { API_BASE, getDiagnostics, getPage, getRun, getTree, stopRun } from '@/lib/api';
import type { Diagnostics, EventFrame, Page, Snapshot, TreeEdge, TreeNode } from '@/lib/types';
import { PageSidebar } from './PageSidebar';
import { PageDetail } from './PageDetail';
import { TreeView } from './TreeView';
import { DiagnosticsPanel } from './DiagnosticsPanel';

type Tab = 'read' | 'graph';
type StreamState = 'live' | 'reconnecting' | 'stopped';

// Page detail load state — loading is derived (no selectedId in data yet)
type PageLoadState = {
  id: string;
  page: Page | null;
  error: string | null;
} | null;

function mergePages(existing: Page[], incoming: Page[]): Page[] {
  const map = new Map(existing.map((p) => [p.id, p]));
  for (const p of incoming) map.set(p.id, p);
  return [
    ...existing.map((p) => map.get(p.id)!),
    ...incoming.filter((p) => !existing.some((e) => e.id === p.id)),
  ];
}

function mergeNodes(existing: TreeNode[], incoming: TreeNode[]): TreeNode[] {
  const ids = new Set(existing.map((n) => n.id));
  return [...existing, ...incoming.filter((n) => !ids.has(n.id))];
}

function mergeEdges(existing: TreeEdge[], incoming: TreeEdge[]): TreeEdge[] {
  const seen = new Set(existing.map((e) => `${e.parent_page_id}:${e.child_page_id}`));
  return [
    ...existing,
    ...incoming.filter((e) => !seen.has(`${e.parent_page_id}:${e.child_page_id}`)),
  ];
}

export function RunWorkspace({ runId, initial }: { runId: string; initial: Snapshot }) {
  const [run, setRun] = useState(initial);
  const [pages, setPages] = useState<Page[]>(initial.pages ?? []);
  const [tree, setTree] = useState<{ nodes: TreeNode[]; edges: TreeEdge[] }>({
    nodes: initial.tree_nodes ?? [],
    edges: initial.tree_edges ?? [],
  });
  const [diagnostics, setDiagnostics] = useState<Diagnostics>(initial.diagnostics);
  const [rootPageId, setRootPageId] = useState<string | undefined>(initial.root_page_id);

  const [selectedId, setSelectedId] = useState<string>(
    initial.pages[0]?.id ?? initial.tree_nodes[0]?.id ?? ''
  );

  // Flash set for newly-arrived pages — updated inside SSE callbacks, not effect body
  const [newPageIds, setNewPageIds] = useState<Set<string>>(new Set());

  // Page detail: loading = selectedId exists but loaded?.id !== selectedId
  const [pageLoad, setPageLoad] = useState<PageLoadState>(
    initial.pages[0]
      ? { id: initial.pages[0].id, page: initial.pages[0], error: null }
      : null
  );
  const pageLoading = Boolean(selectedId && (!pageLoad || pageLoad.id !== selectedId));
  const pageError = pageLoad?.id === selectedId ? pageLoad.error : null;
  const selectedPage = pageLoad?.id === selectedId ? pageLoad.page : null;

  const [stopError, setStopError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>('read');
  const [streamState, setStreamState] = useState<StreamState>(
    initial.status === 'running' ? 'live' : 'stopped'
  );

  // Load page detail when selection changes (no sync setState in effect body)
  useEffect(() => {
    if (!selectedId) return;
    let active = true;
    const id = selectedId;
    getPage(runId, id)
      .then((p) => {
        if (active) setPageLoad({ id, page: p, error: null });
      })
      .catch((err: unknown) => {
        if (active)
          setPageLoad({
            id,
            page: null,
            error: err instanceof Error ? err.message : 'Failed to load page',
          });
      });
    return () => { active = false; };
  }, [runId, selectedId]);

  // SSE subscription
  useEffect(() => {
    const es = new EventSource(`${API_BASE}/runs/${runId}/events`);

    const onFrame = (event: MessageEvent<string>) => {
      const frame = JSON.parse(event.data) as EventFrame;
      setRun((prev) => ({ ...prev, status: frame.status, summary: frame.summary }));

      if (frame.new_pages?.length) {
        const incoming = frame.new_pages;
        setPages((prev) => mergePages(prev, incoming));
        setSelectedId((prev) => prev || incoming[0].id);
        // Flash animation — set inside SSE event callback, not synchronously in effect
        const ids = new Set(incoming.map((p) => p.id));
        setNewPageIds(ids);
        setTimeout(() => setNewPageIds(new Set()), 1000);
      }

      if (frame.tree_nodes?.length || frame.tree_edges?.length) {
        setTree((prev) => ({
          nodes: mergeNodes(prev.nodes, frame.tree_nodes ?? []),
          edges: mergeEdges(prev.edges, frame.tree_edges ?? []),
        }));
      }

      if (frame.status === 'stopped') {
        setStreamState('stopped');
        void Promise.all([getRun(runId), getTree(runId), getDiagnostics(runId)])
          .then(([finalRun, finalTree, finalDiag]) => {
            setRun(finalRun);
            setRootPageId(finalRun.root_page_id);
            setTree(finalTree);
            setDiagnostics(finalDiag);
          })
          .catch(() => {});
      } else {
        setStreamState('live');
      }
    };

    es.addEventListener('frame', onFrame as EventListener);
    es.onerror = () =>
      setStreamState((prev) => (prev === 'stopped' ? prev : 'reconnecting'));

    return () => {
      es.removeEventListener('frame', onFrame as EventListener);
      es.close();
    };
  }, [runId]);

  // Periodic poll fallback (every 12s while running)
  useEffect(() => {
    if (streamState === 'stopped') return;
    const interval = setInterval(() => {
      void Promise.all([getRun(runId), getTree(runId)])
        .then(([r, t]) => {
          setRun((prev) => ({ ...prev, summary: r.summary, status: r.status }));
          setRootPageId(r.root_page_id);
          setTree(t);
          if (r.status !== 'running') setStreamState('stopped');
        })
        .catch(() => {});
    }, 12000);
    return () => clearInterval(interval);
  }, [runId, streamState]);

  const handleStop = useCallback(async () => {
    setStopError(null);
    try {
      await stopRun(runId);
    } catch (err) {
      setStopError(err instanceof Error ? err.message : 'Stop failed');
    }
  }, [runId]);

  const title = useMemo(
    () =>
      run.config.mode === 'keyword'
        ? run.seed.query || run.config.input
        : run.seed.primary_url || run.config.input,
    [run]
  );

  const { summary } = run;

  return (
    <div className="workspace-shell">
      {/* Status bar */}
      <header className="status-bar">
        <Link href="/" className="status-back">← Arachne</Link>
        <h1 className="status-title" title={title}>{title}</h1>
        <div className="status-right">
          <div className="status-stat">
            <span className="status-stat__val">{summary.pages_fetched}</span>
            <span className="status-stat__label">fetched</span>
          </div>
          {summary.pages_failed > 0 && (
            <div className="status-stat">
              <span className="status-stat__val" style={{ color: '#c07070' }}>
                {summary.pages_failed}
              </span>
              <span className="status-stat__label">failed</span>
            </div>
          )}
          <div className="status-divider" />
          <div className={`live-badge live-badge--${streamState}`}>
            <span className="live-badge__dot" />
            {streamState}
          </div>
          {streamState !== 'stopped' && (
            <button className="btn-ghost" type="button" onClick={() => void handleStop()}>
              Stop
            </button>
          )}
        </div>
      </header>

      {stopError && (
        <div className="form-error" style={{ margin: '8px 0' }}>{stopError}</div>
      )}

      {/* Main layout */}
      <div className="workspace-main">
        <PageSidebar pages={pages} selectedId={selectedId} newPageIds={newPageIds} onSelect={setSelectedId} />

        <div className="content-area">
          <div className="content-tabs">
            <button
              type="button"
              className={`content-tab${tab === 'read' ? ' active' : ''}`}
              onClick={() => setTab('read')}
            >
              Read
            </button>
            <button
              type="button"
              className={`content-tab${tab === 'graph' ? ' active' : ''}`}
              onClick={() => setTab('graph')}
            >
              Graph
            </button>
          </div>

          {tab === 'read' ? (
            <PageDetail
              page={selectedPage}
              loading={pageLoading}
              error={pageError}
            />
          ) : (
            <TreeView
              nodes={tree.nodes}
              edges={tree.edges}
              rootPageId={rootPageId}
              selectedId={selectedId}
              onSelect={(id) => { setSelectedId(id); setTab('read'); }}
            />
          )}
        </div>
      </div>

      {/* Diagnostics footer */}
      <DiagnosticsPanel
        summary={summary}
        seed={run.seed}
        diagnostics={diagnostics}
      />
    </div>
  );
}
