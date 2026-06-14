import React from 'react';
import './RadarChart.css';

const polarToXY = (cx, cy, radius, angleDeg) => {
  const rad = ((angleDeg - 90) * Math.PI) / 180;
  return [cx + radius * Math.cos(rad), cy + radius * Math.sin(rad)];
};

const truncate = (text, max = 18) => {
  const value = String(text ?? '');
  return value.length > max ? `${value.slice(0, max - 1)}…` : value;
};

const RadarChart = ({ data = [], size = 340, levels = 4 }) => {
  const axes = (Array.isArray(data) ? data : []).filter(Boolean);
  const n = axes.length;

  if (n === 0) {
    return <p className="radar-empty">Нет предметов для построения диаграммы.</p>;
  }

  const clampPercent = (value) => Math.max(0, Math.min(100, Number(value) || 0));

  if (n < 3) {
    const circleSize = 130;
    const stroke = 10;
    const circleRadius = (circleSize - stroke) / 2;
    const circumference = 2 * Math.PI * circleRadius;

    return (
      <div className="radar-circles">
        {axes.map((axis, i) => {
          const percent = clampPercent(axis.percent);
          const offset = circumference * (1 - percent / 100);
          return (
            <div className="radar-circle" key={axis.subject_id ?? i}>
              <svg width={circleSize} height={circleSize} viewBox={`0 0 ${circleSize} ${circleSize}`}>
                <circle
                  className="radar-circle-track"
                  cx={circleSize / 2}
                  cy={circleSize / 2}
                  r={circleRadius}
                  strokeWidth={stroke}
                  fill="none"
                />
                <circle
                  className="radar-circle-fill"
                  cx={circleSize / 2}
                  cy={circleSize / 2}
                  r={circleRadius}
                  strokeWidth={stroke}
                  fill="none"
                  strokeDasharray={circumference}
                  strokeDashoffset={offset}
                  transform={`rotate(-90 ${circleSize / 2} ${circleSize / 2})`}
                />
                <text x="50%" y="50%" textAnchor="middle" dominantBaseline="middle" className="radar-circle-pct">
                  {Math.round(percent)}%
                </text>
              </svg>
              <div className="radar-circle-label">{axis.subject_name}</div>
              <div className="radar-circle-score">{axis.score ?? 0} / {axis.max_score ?? 0}</div>
            </div>
          );
        })}
      </div>
    );
  }

  const padding = 70;
  const cx = size / 2;
  const cy = size / 2;
  const radius = size / 2 - padding;
  const angleFor = (i) => (360 / n) * i;

  const rings = [];
  for (let level = 1; level <= levels; level += 1) {
    const r = (radius * level) / levels;
    rings.push(axes.map((_, i) => polarToXY(cx, cy, r, angleFor(i)).join(',')).join(' '));
  }

  const dataPoints = axes.map((axis, i) => {
    const r = (radius * clampPercent(axis.percent)) / 100;
    return polarToXY(cx, cy, r, angleFor(i));
  });
  const dataPolygon = dataPoints.map((point) => point.join(',')).join(' ');

  return (
    <div className="radar-chart">
      <svg viewBox={`0 0 ${size} ${size}`} className="radar-svg" role="img" aria-label="Диаграмма успеваемости">
        {rings.map((points, index) => (
          <polygon key={`ring-${index}`} className="radar-ring" points={points} />
        ))}

        {axes.map((axis, i) => {
          const [x, y] = polarToXY(cx, cy, radius, angleFor(i));
          const [lx, ly] = polarToXY(cx, cy, radius + 16, angleFor(i));
          const anchor = Math.abs(lx - cx) < 1 ? 'middle' : lx > cx ? 'start' : 'end';
          return (
            <g key={`axis-${axis.subject_id ?? i}`}>
              <line className="radar-axis" x1={cx} y1={cy} x2={x} y2={y} />
              <text className="radar-label" x={lx} y={ly} textAnchor={anchor} dominantBaseline="middle">
                <title>{axis.subject_name}</title>
                {truncate(axis.subject_name)}
              </text>
              <text className="radar-label-pct" x={lx} y={ly + 14} textAnchor={anchor} dominantBaseline="middle">
                {Math.round(clampPercent(axis.percent))}%
              </text>
            </g>
          );
        })}

        <polygon className="radar-area" points={dataPolygon} />
        {dataPoints.map((point, i) => (
          <circle key={`dot-${axes[i].subject_id ?? i}`} className="radar-dot" cx={point[0]} cy={point[1]} r={3.5}>
            <title>
              {`${axes[i].subject_name}: ${axes[i].score ?? 0}/${axes[i].max_score ?? 0} (${Math.round(
                clampPercent(axes[i].percent)
              )}%)`}
            </title>
          </circle>
        ))}
      </svg>
    </div>
  );
};

export default RadarChart;
