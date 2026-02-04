'use client';

import { useEffect, useRef } from 'react';

type Edge = { src: string; dst: string; count: number };

export function GraphCanvas({ nodes, edges }: { nodes: string[]; edges: Edge[] }) {
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

    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = 'rgba(225, 93, 59, 0.06)';
    ctx.fillRect(0, 0, width, height);

    const centerX = width / 2;
    const centerY = height / 2;
    const radius = Math.min(width, height) * 0.38;

    const positions = new Map<string, { x: number; y: number }>();
    const fontFamily =
      getComputedStyle(document.documentElement).getPropertyValue('--font-body') || 'sans-serif';

    nodes.forEach((host) => {
      const h = hash(host);
      const angle = (h % 360) * (Math.PI / 180);
      const r = radius * (0.4 + ((h % 100) / 100) * 0.6);
      const x = centerX + Math.cos(angle) * r;
      const y = centerY + Math.sin(angle) * r;
      positions.set(host, { x, y });
    });

    ctx.strokeStyle = 'rgba(15, 123, 108, 0.25)';
    ctx.lineWidth = 1.2 * dpr;
    edges.forEach((edge) => {
      const from = positions.get(edge.src);
      const to = positions.get(edge.dst);
      if (!from || !to) return;
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(to.x, to.y);
      ctx.stroke();
    });

    nodes.forEach((host) => {
      const pos = positions.get(host);
      if (!pos) return;
      ctx.fillStyle = 'rgba(225, 93, 59, 0.9)';
      ctx.beginPath();
      ctx.arc(pos.x, pos.y, 4 * dpr, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = '#1b1812';
      ctx.font = `${10 * dpr}px ${fontFamily}`;
      ctx.fillText(host, pos.x + 6 * dpr, pos.y - 6 * dpr);
    });
  }, [nodes, edges]);

  return <canvas ref={canvasRef} className="graph-canvas" />;
}

function hash(input: string): number {
  let h = 2166136261;
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return Math.abs(h);
}
