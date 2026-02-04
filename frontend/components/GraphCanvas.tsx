'use client';

import { useEffect, useRef } from 'react';

type Edge = { src: string; dst: string; count: number };

interface GraphCanvasProps {
  nodes: string[];
  edges: Edge[];
}

export function GraphCanvas({ nodes, edges }: GraphCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const width = canvas.clientWidth * dpr;
    const height = canvas.clientHeight * dpr;

    canvas.width = width;
    canvas.height = height;

    // Clear with background
    ctx.fillStyle = '#f4f3f0';
    ctx.fillRect(0, 0, width, height);

    if (nodes.length === 0) {
      ctx.fillStyle = '#999999';
      ctx.font = `${14 * dpr}px system-ui, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText('Waiting for hosts...', width / 2, height / 2);
      return;
    }

    const centerX = width / 2;
    const centerY = height / 2;
    const maxRadius = Math.min(width, height) * 0.4;

    // Calculate positions using golden angle for even distribution
    const positions = new Map<string, { x: number; y: number }>();
    const goldenAngle = Math.PI * (3 - Math.sqrt(5));

    nodes.forEach((host, index) => {
      const angle = index * goldenAngle;
      const distFactor = 0.3 + (index / Math.max(nodes.length, 1)) * 0.7;
      const r = maxRadius * distFactor;
      const x = centerX + Math.cos(angle) * r;
      const y = centerY + Math.sin(angle) * r;
      positions.set(host, { x, y });
    });

    // Draw edges
    edges.forEach((edge) => {
      const from = positions.get(edge.src);
      const to = positions.get(edge.dst);
      if (!from || !to) return;

      const intensity = Math.min(edge.count / 10, 1);
      ctx.strokeStyle = `rgba(224, 122, 95, ${0.15 + intensity * 0.25})`;
      ctx.lineWidth = (1 + intensity) * dpr;
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(to.x, to.y);
      ctx.stroke();
    });

    // Draw nodes
    nodes.forEach((host) => {
      const pos = positions.get(host);
      if (!pos) return;

      // Node circle
      ctx.fillStyle = '#e07a5f';
      ctx.beginPath();
      ctx.arc(pos.x, pos.y, 4 * dpr, 0, Math.PI * 2);
      ctx.fill();

      // Label
      ctx.fillStyle = '#1a1a1a';
      ctx.font = `${10 * dpr}px system-ui, sans-serif`;
      ctx.textAlign = 'left';
      ctx.textBaseline = 'middle';

      const displayHost = host.length > 18 ? host.slice(0, 15) + '...' : host;
      ctx.fillText(displayHost, pos.x + 8 * dpr, pos.y);
    });
  }, [nodes, edges]);

  return (
    <div className="graph-wrapper">
      <canvas ref={canvasRef} className="graph-canvas" />
    </div>
  );
}
