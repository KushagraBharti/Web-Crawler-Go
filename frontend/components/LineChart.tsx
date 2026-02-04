'use client';

import { useMemo } from 'react';

interface LineChartProps {
  data: number[];
  label: string;
  color: string;
}

export function LineChart({ data, label, color }: LineChartProps) {
  const { points, areaPoints, currentValue } = useMemo(() => {
    if (data.length === 0) {
      return { points: '', areaPoints: '', currentValue: 0 };
    }

    const max = Math.max(...data, 1);
    const min = Math.min(...data, 0);
    const range = max - min || 1;
    const current = data[data.length - 1] || 0;

    const linePoints = data
      .map((value, index) => {
        const x = (index / Math.max(data.length - 1, 1)) * 100;
        const y = 100 - ((value - min) / range) * 85 - 7.5;
        return `${x},${y}`;
      })
      .join(' ');

    const area = `0,100 ${linePoints} 100,100`;

    return { points: linePoints, areaPoints: area, currentValue: current };
  }, [data]);

  return (
    <div className="chart-container">
      <div className="chart-label">
        <span>{label}</span>
        {data.length > 0 && (
          <span className="chart-label__value">{currentValue.toFixed(1)}</span>
        )}
      </div>
      <svg viewBox="0 0 100 100" preserveAspectRatio="none" className="chart-svg">
        {points && (
          <>
            <polygon points={areaPoints} fill={color} fillOpacity="0.1" />
            <polyline
              fill="none"
              stroke={color}
              strokeWidth="2"
              strokeLinejoin="round"
              strokeLinecap="round"
              points={points}
            />
          </>
        )}
        {!points && (
          <text x="50" y="50" textAnchor="middle" fill="currentColor" fillOpacity="0.3" fontSize="8">
            Waiting for data...
          </text>
        )}
      </svg>
    </div>
  );
}
