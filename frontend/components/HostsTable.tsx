'use client';

import { Frame } from '@/lib/types';

interface HostsTableProps {
  hosts: Frame['hosts'];
}

export function HostsTable({ hosts }: HostsTableProps) {
  const getErrorClass = (rate: number) => {
    if (rate > 0.2) return 'data-table__cell--danger';
    if (rate > 0.1) return 'data-table__cell--warning';
    return '';
  };

  const getLatencyClass = (ms: number) => {
    if (ms > 2000) return 'data-table__cell--danger';
    if (ms > 1000) return 'data-table__cell--warning';
    return '';
  };

  const getReuseClass = (rate: number) => {
    if (rate > 0.7) return 'data-table__cell--success';
    return '';
  };

  return (
    <section className="panel">
      <span className="badge badge--warning">Telemetry</span>
      <h3 style={{ marginTop: '1rem' }}>Host Metrics</h3>
      <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
        {hosts.length} active hosts
      </p>

      <div className="data-table">
        <div className="data-table__header">
          <span>Host</span>
          <span>Inflight</span>
          <span>P95</span>
          <span>Errors</span>
          <span>Reuse</span>
          <span>Robots</span>
          <span>Circuit</span>
        </div>

        {hosts.length === 0 ? (
          <div className="empty-state">No host data yet</div>
        ) : (
          hosts.map((host) => (
            <div className="data-table__row" key={host.host}>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={host.host}>
                {host.host}
              </span>
              <span>{host.inflight}</span>
              <span className={getLatencyClass(host.p95_ms)}>{host.p95_ms} ms</span>
              <span className={getErrorClass(host.error_rate)}>{(host.error_rate * 100).toFixed(1)}%</span>
              <span className={getReuseClass(host.reuse_rate)}>{(host.reuse_rate * 100).toFixed(0)}%</span>
              <span>{host.robots_state || '—'}</span>
              <span>{host.circuit_state || 'closed'}</span>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
