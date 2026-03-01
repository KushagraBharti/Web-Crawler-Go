'use client';

import { Frame } from '@/lib/types';

interface HostsTableProps {
  hosts: Frame['hosts'];
}

export function HostsTable({ hosts }: HostsTableProps) {
  const getErrorClass = (rate: number) => {
    if (rate > 0.2) return 'cell--danger';
    if (rate > 0.1) return 'cell--warning';
    return '';
  };

  const getLatencyClass = (ms: number) => {
    if (ms > 2000) return 'cell--danger';
    if (ms > 1000) return 'cell--warning';
    return '';
  };

  const getReuseClass = (rate: number) => {
    if (rate > 0.7) return 'cell--success';
    return '';
  };

  return (
    <section className="panel" aria-labelledby="hosts-table-heading">
      <span className="badge badge--warning">Telemetry</span>
      <h3 id="hosts-table-heading" style={{ marginTop: '1rem' }}>Host Metrics</h3>
      <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
        {hosts.length} active hosts
      </p>

      {hosts.length === 0 ? (
        <div className="empty-state" role="status">
          <div className="skeleton skeleton--table-row" />
          <div className="skeleton skeleton--table-row" />
          <div className="skeleton skeleton--table-row" />
          <p style={{ marginTop: '1rem' }}>Waiting for host data...</p>
        </div>
      ) : (
        <div className="table-scroll" role="region" aria-label="Host metrics table" tabIndex={0}>
          <table className="semantic-table" aria-labelledby="hosts-table-heading">
            <thead>
              <tr>
                <th scope="col">Host</th>
                <th scope="col">Inflight</th>
                <th scope="col">P95</th>
                <th scope="col">Errors</th>
                <th scope="col">Reuse</th>
                <th scope="col">Robots</th>
                <th scope="col">Circuit</th>
              </tr>
            </thead>
            <tbody aria-live="polite" aria-relevant="additions removals">
              {hosts.map((host) => (
                <tr key={host.host}>
                  <td title={host.host} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '200px' }}>
                    {host.host}
                  </td>
                  <td>{host.inflight}</td>
                  <td className={getLatencyClass(host.p95_ms)}>{host.p95_ms} ms</td>
                  <td className={getErrorClass(host.error_rate)}>{(host.error_rate * 100).toFixed(1)}%</td>
                  <td className={getReuseClass(host.reuse_rate)}>{(host.reuse_rate * 100).toFixed(0)}%</td>
                  <td>{host.robots_state || '\u2014'}</td>
                  <td>{host.circuit_state || 'closed'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
