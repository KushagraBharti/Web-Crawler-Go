'use client';

import type { TreeEdge, TreeNode } from '@/lib/types';

type Placed = { x: number; y: number; node: TreeNode };

const COL_W = 220;   // horizontal space per depth level
const NODE_W = 120;  // node rect width
const NODE_H = 56;   // node rect height
const LEAF_H = 88;   // vertical space allocated per leaf
const MARGIN_X = 80; // left margin
const MARGIN_Y = 48; // top/bottom margin

export function TreeView({ nodes, edges, selectedId, onSelect }: {
  nodes: TreeNode[];
  edges: TreeEdge[];
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  if (nodes.length === 0) {
    return <div className="empty-panel">Tree will appear here as pages are discovered.</div>;
  }

  // O(1) node lookup
  const nodeById = new Map(nodes.map((n) => [n.id, n]));

  // Build children map from edges
  const childrenMap = new Map<string, string[]>();
  for (const n of nodes) childrenMap.set(n.id, []);
  for (const e of edges) {
    const kids = childrenMap.get(e.parent_page_id);
    if (kids) kids.push(e.child_page_id);
  }

  // Find root: prefer depth 0, fallback to node not appearing as a child
  const childIds = new Set(edges.map((e) => e.child_page_id));
  const root =
    nodes.find((n) => n.depth === 0) ||
    nodes.find((n) => !childIds.has(n.id)) ||
    nodes[0];

  // Count leaf nodes in a subtree (leaves = nodes with no children)
  function countLeaves(id: string): number {
    const kids = childrenMap.get(id) || [];
    if (kids.length === 0) return 1;
    return kids.reduce((s, k) => s + countLeaves(k), 0);
  }

  // Place nodes recursively. x = center-x, y = center-y.
  // Returns the next available yOffset after this subtree.
  const placed = new Map<string, Placed>();

  function place(id: string, yOffset: number, depth: number): number {
    const node = nodeById.get(id);
    if (!node) return yOffset;
    const kids = childrenMap.get(id) || [];

    if (kids.length === 0) {
      placed.set(id, { x: MARGIN_X + depth * COL_W, y: yOffset + NODE_H / 2, node });
      return yOffset + LEAF_H;
    }

    let cursor = yOffset;
    for (const kid of kids) {
      cursor = place(kid, cursor, depth + 1);
    }

    const firstKid = placed.get(kids[0]);
    const lastKid = placed.get(kids[kids.length - 1]);
    const centerY =
      firstKid && lastKid ? (firstKid.y + lastKid.y) / 2 : yOffset + NODE_H / 2;

    placed.set(id, { x: MARGIN_X + depth * COL_W, y: centerY, node });
    return cursor;
  }

  let nextY = place(root.id, MARGIN_Y, 0);

  // Place any orphaned nodes not reachable from root
  for (const n of nodes) {
    if (!placed.has(n.id)) {
      placed.set(n.id, { x: MARGIN_X + n.depth * COL_W, y: nextY + NODE_H / 2, node: n });
      nextY += LEAF_H;
    }
  }

  const allPlaced = Array.from(placed.values());
  const maxX = Math.max(...allPlaced.map((p) => p.x));
  const maxY = Math.max(...allPlaced.map((p) => p.y));
  const svgW = Math.max(maxX + NODE_W / 2 + MARGIN_X, 720);
  const svgH = maxY + NODE_H / 2 + MARGIN_Y;

  return (
    <div className="tree-shell">
      <svg
        className="tree-svg"
        viewBox={`0 0 ${svgW} ${svgH}`}
        style={{ height: svgH }}
        width="100%"
      >
        <defs>
          <linearGradient id="treeEdge" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="rgba(246, 118, 37, 0.8)" />
            <stop offset="100%" stopColor="rgba(41, 113, 255, 0.65)" />
          </linearGradient>
        </defs>

        {edges.map((edge) => {
          const from = placed.get(edge.parent_page_id);
          const to = placed.get(edge.child_page_id);
          if (!from || !to) return null;
          const midX = (from.x + to.x) / 2;
          return (
            <path
              key={`${edge.parent_page_id}-${edge.child_page_id}`}
              d={`M ${from.x + NODE_W / 2} ${from.y} C ${midX} ${from.y}, ${midX} ${to.y}, ${to.x - NODE_W / 2} ${to.y}`}
              fill="none"
              stroke="url(#treeEdge)"
              strokeWidth="3"
              strokeLinecap="round"
            />
          );
        })}

        {allPlaced.map(({ x, y, node }) => (
          <g
            className={selectedId === node.id ? 'tree-node active' : 'tree-node'}
            key={node.id}
            onClick={() => onSelect(node.id)}
            transform={`translate(${x - NODE_W / 2}, ${y - NODE_H / 2})`}
          >
            <rect rx="18" ry="18" width={NODE_W} height={NODE_H} />
            <text x="12" y="22">{clip(node.title || node.url, 18)}</text>
            <text className="tree-node__meta" x="12" y="40">
              Depth {node.depth}
            </text>
          </g>
        ))}
      </svg>
    </div>
  );
}

function clip(value: string, max: number) {
  return value.length > max ? `${value.slice(0, max - 1)}…` : value;
}
