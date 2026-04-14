'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { API_BASE, getDiagnostics, getPage, getRun, getTree } from '@/lib/api';
import type { Diagnostics, EventFrame, Page, Snapshot, TreeEdge, TreeNode } from '@/lib/types';
import { PageDetail } from './PageDetail';
import { PageSidebar } from './PageSidebar';
import { TreeView } from './TreeView';
import { DiagnosticsPanel } from './DiagnosticsPanel';

export function RunWorkspace({ runId, initial }: { runId: string; initial: Snapshot }) {
  const [run, setRun] = useState(initial);
  const [pages, setPages] = useState(initial.pages);
  const [selectedId, setSelectedId] = useState(initial.pages[0]?.id || initial.tree_nodes[0]?.id || '');
  const [selectedPage, setSelectedPage] = useState<Page | null>(initial.pages[0] || null);
  const [tree, setTree] = useState<{ nodes: TreeNode[]; edges: TreeEdge[] }>({
    nodes: initial.tree_nodes,
    edges: initial.tree_edges
  });
  const [diagnostics, setDiagnostics] = useState<Diagnostics>(initial.diagnostics);
  const [streamState, setStreamState] = useState<'live' | 'reconnecting' | 'stopped'>(
    initial.status === 'running' ? 'live' : 'stopped'
  );

  // Ref to read latest selectedId in EventSource handler without re-subscribing
  const selectedIdRef = useRef(selectedId);
  useEffect(() => {
    selectedIdRef.current = selectedId;
  }, [selectedId]);

  // Fetch full page detail when selection changes
  useEffect(() => {
    if (!selectedId) return;
    void getPage(runId, selectedId).then(setSelectedPage).catch(() => undefined);
  }, [runId, selectedId]);

  // Single EventSource for all real-time updates.
  // On stop: do one final fetch to capture complete state.
  useEffect(() => {
    const eventSource = new EventSource(`${API_BASE}/runs/${runId}/events`);

    const onFrame = (event: MessageEvent<string>) => {
      const frame = JSON.parse(event.data) as EventFrame;

      setRun((prev) => ({ ...prev, status: frame.status, summary: frame.summary }));

      if (frame.new_pages?.length) {
        setPages((prev) => mergePages(prev, frame.new_pages || []));
        const latest = frame.latest || frame.new_pages[frame.new_pages.length - 1];
        if (!selectedIdRef.current) {
          setSelectedId(latest.id);
        }
      }

      if (frame.tree_nodes?.length || frame.tree_edges?.length) {
        setTree((prev) => ({
          nodes: mergeNodes(prev.nodes, frame.tree_nodes || []),
          edges: mergeEdges(prev.edges, frame.tree_edges || [])
        }));
      }

      if (frame.status === 'stopped') {
        setStreamState('stopped');
        // One-shot final sync to capture any state the stream may have missed
        void Promise.all([getRun(runId), getTree(runId), getDiagnostics(runId)])
          .then(([finalRun, finalTree, finalDiag]) => {
            setRun(finalRun);
            setTree(finalTree);
            setDiagnostics(finalDiag);
          })
          .catch(() => undefined);
      } else {
        setStreamState('live');
      }
    };

    eventSource.addEventListener('frame', onFrame as EventListener);
    eventSource.onerror = () =>
      setStreamState((prev) => (prev === 'stopped' ? prev : 'reconnecting'));

    return () => {
      eventSource.removeEventListener('frame', onFrame as EventListener);
      eventSource.close();
    };
  }, [runId]);

  const activePage = useMemo(
    () => selectedPage || pages.find((page) => page.id === selectedId) || null,
    [pages, selectedId, selectedPage]
  );

  return (
    <main className="workspace">
      <section className="workspace-hero">
        <div>
          <div className="eyebrow">Run workspace</div>
          <h1>{run.config.mode === 'keyword' ? run.seed.query : run.seed.primary_url}</h1>
          <p>Rooted crawl tree, readable page content, and local JSON diagnostics.</p>
        </div>
        <div className="hero-status">
          <span className={`pill pill--${streamState}`}>{streamState}</span>
          <span className="run-id">{runId}</span>
        </div>
      </section>

      <DiagnosticsPanel diagnostics={diagnostics} seed={run.seed} summary={run.summary} />

      <section className="workspace-grid">
        <PageSidebar pages={pages} selectedId={selectedId} onSelect={setSelectedId} />
        <PageDetail page={activePage} />
      </section>

      <section className="panel">
        <div className="panel-header-block">
          <div className="eyebrow">Crawl tree</div>
          <h3>Parent-child discovery flow</h3>
        </div>
        <TreeView
          nodes={tree.nodes}
          edges={tree.edges}
          onSelect={setSelectedId}
          selectedId={selectedId}
        />
      </section>
    </main>
  );
}

function mergePages(existing: Page[], next: Page[]) {
  const map = new Map(existing.map((page) => [page.id, page]));
  next.forEach((page) => map.set(page.id, page));
  return Array.from(map.values()).sort((a, b) =>
    a.discovered_at.localeCompare(b.discovered_at)
  );
}

function mergeNodes(existing: TreeNode[], next: TreeNode[]) {
  const map = new Map(existing.map((node) => [node.id, node]));
  next.forEach((node) => map.set(node.id, node));
  return Array.from(map.values());
}

function mergeEdges(existing: TreeEdge[], next: TreeEdge[]) {
  const map = new Map(
    existing.map((edge) => [`${edge.parent_page_id}-${edge.child_page_id}`, edge])
  );
  next.forEach((edge) =>
    map.set(`${edge.parent_page_id}-${edge.child_page_id}`, edge)
  );
  return Array.from(map.values());
}
