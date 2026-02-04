'use client';

import { Frame } from '@/lib/types';

export function HostsTable({ hosts }: { hosts: Frame['hosts'] }) {
  return (
    <div className="panel">
      <div className="badge">Per-host telemetry</div>
      <h2 style={{ marginTop: 12 }}>Host behavior</h2>
      <div className="table">
        <div className="table-row table-head">
          <span>Host</span>
          <span>Inflight</span>
          <span>P95</span>
          <span>Errors</span>
          <span>Reuse</span>
          <span>Robots</span>
          <span>Circuit</span>
        </div>
        {hosts.map((h) => (
          <div className="table-row" key={h.host}>
            <span>{h.host}</span>
            <span>{h.inflight}</span>
            <span>{h.p95_ms} ms</span>
            <span>{Math.round(h.error_rate * 100)}%</span>
            <span>{Math.round(h.reuse_rate * 100)}%</span>
            <span>{h.robots_state || 'n/a'}</span>
            <span>{h.circuit_state || 'closed'}</span>
          </div>
        ))}
      </div>
    </div>
  );
}