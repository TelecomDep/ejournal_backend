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
    const pathParts = [];
    for (let row = 0; row < count; row += 1) {
      let runStart = -1;
      for (let col = 0; col <= count; col += 1) {
        const isDark = col < count && qr.isDark(row, col);
        if (isDark && runStart < 0) {
          runStart = col;
        } else if (!isDark && runStart >= 0) {
          const runWidth = col - runStart;
          pathParts.push(`M${runStart} ${row}h${runWidth}v1h-${runWidth}z`);
          runStart = -1;
        }
      }
    }
    return { count, path: pathParts.join('') };
  }, [value]);

  if (!modules) {
    return null;
  }

  const { count, path } = modules;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${count} ${count}`}
      shapeRendering="crispEdges"
      role="img"
      aria-label="QR код для ссылки"
    >
      <rect width={count} height={count} fill="#ffffff" />
      <path d={path} fill="#000000" />
    </svg>
  );
}
