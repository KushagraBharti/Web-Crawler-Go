'use client';

import type { TreeEdge, TreeNode } from '@/lib/types';

const COL_W  = 200;
const NODE_W = 128;
const NODE_H = 52;
const LEAF_H = 80;
const PAD_X  = 60;
const PAD_Y  = 40;

type Placed = { x: number; y: number; node: TreeNode };

function clip(s: string, max: number): string {
  return s.length > max ? s.slice(0, max - 1) + '…' : s;
}

export function TreeView({
  nodes,
  edges,
  rootPageId,
  selectedId,
  onSelect,
}: {
  nodes: TreeNode[];
  edges: TreeEdge[];
  rootPageId?: string;
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  if (nodes.length === 0) {
    return (
      <div className="tree-pane">
        <div className="tree-empty">Tree builds as pages are discovered</div>
      </div>
    );
  }

  const nodeById = new Map(nodes.map((n) => [n.id, n]));

  // Build children map strictly from edges (parent → children)
  const childrenOf = new Map<string, string[]>();
  for (const n of nodes) childrenOf.set(n.id, []);
  for (const e of edges) {
    if (nodeById.has(e.parent_page_id) && nodeById.has(e.child_page_id)) {
      childrenOf.get(e.parent_page_id)?.push(e.child_page_id);
    }
  }

  // Determine root — prefer explicit rootPageId, then depth-0, then first node
  const childIds = new Set(edges.map((e) => e.child_page_id));
  const root =
    (rootPageId && nodeById.get(rootPageId)) ||
    nodes.find((n) => n.depth === 0) ||
    nodes.find((n) => !childIds.has(n.id)) ||
    nodes[0];

  // Count leaves in subtree (for vertical space allocation)
  function countLeaves(id: string): number {
    const kids = childrenOf.get(id) ?? [];
    if (kids.length === 0) return 1;
    return kids.reduce((s, k) => s + countLeaves(k), 0);
  }

  // Place nodes recursively: x = depth column, y = center of children band
  const placed = new Map<string, Placed>();

  function place(id: string, yOffset: number, depth: number): number {
    const node = nodeById.get(id);
    if (!node || placed.has(id)) return yOffset;
    const kids = childrenOf.get(id) ?? [];

    if (kids.length === 0) {
      placed.set(id, { x: PAD_X + depth * COL_W, y: yOffset + NODE_H / 2, node });
      return yOffset + LEAF_H;
    }

    let cursor = yOffset;
    for (const kid of kids) {
      cursor = place(kid, cursor, depth + 1);
    }

    const first = placed.get(kids[0]);
    const last  = placed.get(kids[kids.length - 1]);
    const cy =
      first && last ? (first.y + last.y) / 2 : yOffset + NODE_H / 2;

    placed.set(id, { x: PAD_X + depth * COL_W, y: cy, node });
    return cursor;
  }

  let nextY = place(root.id, PAD_Y, 0);

  // Orphaned nodes not reachable from root
  for (const n of nodes) {
    if (!placed.has(n.id)) {
      placed.set(n.id, {
        x: PAD_X + n.depth * COL_W,
        y: nextY + NODE_H / 2,
        node: n,
      });
      nextY += LEAF_H;
    }
  }

  const all   = Array.from(placed.values());
  const maxX  = Math.max(...all.map((p) => p.x));
  const maxY  = Math.max(...all.map((p) => p.y));
  const svgW  = Math.max(maxX + NODE_W + PAD_X, 600);
  const svgH  = maxY + NODE_H / 2 + PAD_Y;

  return (
    <div className="tree-pane">
      <svg
        className="tree-svg"
        viewBox={`0 0 ${svgW} ${svgH}`}
        width={svgW}
        height={svgH}
      >
        {/* Edges — draw only when both endpoints are placed */}
        {edges.map((edge) => {
          const from = placed.get(edge.parent_page_id);
          const to   = placed.get(edge.child_page_id);
          if (!from || !to) return null;
          const fx = from.x + NODE_W;
          const fy = from.y;
          const tx = to.x;
          const ty = to.y;
          const mx = (fx + tx) / 2;
          return (
            <path
              key={`${edge.parent_page_id}→${edge.child_page_id}`}
              className="tree-edge-path"
              d={`M ${fx} ${fy} C ${mx} ${fy}, ${mx} ${ty}, ${tx} ${ty}`}
            />
          );
        })}

        {/* Nodes */}
        {all.map(({ x, y, node }) => {
          const isRoot     = node.id === root.id;
          const isSelected = node.id === selectedId;
          const cx = [
            'tree-node',
            isRoot     ? 'tree-node--root'     : '',
            isSelected ? 'tree-node--selected' : '',
          ].filter(Boolean).join(' ');

          return (
            <g
              key={node.id}
              className={cx}
              transform={`translate(${x}, ${y - NODE_H / 2})`}
              onClick={() => onSelect(node.id)}
            >
              <rect
                className="tree-node__rect"
                width={NODE_W}
                height={NODE_H}
                rx={3}
              />
              {isRoot && (
                <text
                  className="tree-node__seed"
                  x={10}
                  y={13}
                >
                  SEED
                </text>
              )}
              <text
                className="tree-node__title"
                x={10}
                y={isRoot ? 27 : 20}
              >
                {clip(node.title || node.url, 16)}
              </text>
              <text
                className="tree-node__depth"
                x={10}
                y={isRoot ? 40 : 34}
              >
                Depth {node.depth}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}
