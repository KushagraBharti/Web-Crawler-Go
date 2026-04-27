import type { Diagnostics, Page, Snapshot, TreeEdge, TreeNode } from './types';

export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options?.headers || {}) },
    cache: 'no-store',
    ...options
  });
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    try {
      const parsed = JSON.parse(text) as { error?: string };
      message = parsed.error || text;
    } catch {}
    throw new Error(message || `Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function getRun(runId: string) {
  return fetchJSON<Snapshot>(`/runs/${runId}`);
}

export async function getPages(runId: string) {
  return fetchJSON<{ items: Page[] }>(`/runs/${runId}/pages`);
}

export async function getPage(runId: string, pageId: string) {
  return fetchJSON<Page>(`/runs/${runId}/pages/${pageId}`);
}

export async function getTree(runId: string) {
  return fetchJSON<{ nodes: TreeNode[]; edges: TreeEdge[] }>(`/runs/${runId}/tree`);
}

export async function getDiagnostics(runId: string) {
  return fetchJSON<Diagnostics>(`/runs/${runId}/diagnostics`);
}

export async function stopRun(runId: string) {
  return fetchJSON<{ status: string }>(`/runs/${runId}/stop`, { method: 'POST' });
}

export async function startRun(runId: string, seedUrl?: string) {
  return fetchJSON<{ status: string }>(`/runs/${runId}/start`, {
    method: 'POST',
    body: seedUrl ? JSON.stringify({ seed_url: seedUrl }) : undefined,
  });
}
