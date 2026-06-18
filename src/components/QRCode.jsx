import React, { useMemo } from 'react';
import qrcodegen from '../utils/qrcodeGenerator';

export default function QRCode({ value, size = 160 }) {
  const modules = useMemo(() => {
    if (!value) {
      return null;
    }
    const qr = qrcodegen(0, 'M');
    qr.addData(value);
    qr.make();
    const count = qr.getModuleCount();
    const cells = [];
    for (let row = 0; row < count; row += 1) {
      for (let col = 0; col < count; col += 1) {
        if (qr.isDark(row, col)) {
          cells.push([row, col]);
        }
      }
    }
    return { count, cells };
  }, [value]);

  if (!modules) {
    return null;
  }

  const { count, cells } = modules;
  const cellSize = size / count;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      role="img"
      aria-label="QR код для ссылки"
    >
      <rect width={size} height={size} fill="#ffffff" />
      {cells.map(([row, col]) => (
        <rect
          key={`${row}-${col}`}
          x={col * cellSize}
          y={row * cellSize}
          width={cellSize}
          height={cellSize}
          fill="#000000"
        />
      ))}
    </svg>
  );
}
