import { create } from 'zustand';
import { Frame } from './types';

const SERIES_LIMIT = 60;

type Edge = { src: string; dst: string; count: number };

type RunState = {
  throughput: number[];
  frontier: number[];
  fetch: number[];
  parse: number[];
  errors: Frame['errors'];
  hosts: Frame['hosts'];
  nodes: string[];
  edges: Edge[];
  lastUpdated?: string;
  applyFrame: (frame: Frame) => void;
};

export const useRunStore = create<RunState>((set, get) => {
  const nodeSet = new Set<string>();
  const edgeMap = new Map<string, Edge>();

  const trim = (arr: number[]) => (arr.length > SERIES_LIMIT ? arr.slice(-SERIES_LIMIT) : arr);

  return {
    throughput: [],
    frontier: [],
    fetch: [],
    parse: [],
    errors: [],
    hosts: [],
    nodes: [],
    edges: [],
    applyFrame: (frame: Frame) => {
      const nextThroughput = trim([...get().throughput, frame.throughput.pages_per_sec]);
      const nextFrontier = trim([...get().frontier, frame.queues.frontier]);
      const nextFetch = trim([...get().fetch, frame.queues.fetch]);
      const nextParse = trim([...get().parse, frame.queues.parse]);

      frame.graph_delta.nodes.forEach((n) => nodeSet.add(n));
      frame.graph_delta.edges.forEach(([src, dst, count]) => {
        const key = `${src}->${dst}`;
        const existing = edgeMap.get(key);
        if (existing) {
          existing.count += count;
        } else {
          edgeMap.set(key, { src, dst, count });
        }
      });

      set({
        throughput: nextThroughput,
        frontier: nextFrontier,
        fetch: nextFetch,
        parse: nextParse,
        errors: frame.errors,
        hosts: frame.hosts,
        nodes: Array.from(nodeSet),
        edges: Array.from(edgeMap.values()),
        lastUpdated: frame.ts
      });
    }
  };
});