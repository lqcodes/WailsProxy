import React, { useEffect, useRef } from 'react';
import { useTrafficStore } from '../stores/trafficStore';
import { useThemeStore, COLOR_MAP } from '../stores/themeStore';
import { formatSpeed } from '../utils/format';

export const TrafficChart: React.FC = () => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  // 精确订阅 history 分片，避免引起页面其它区域重渲染
  const history = useTrafficStore((s) => s.history);
  const current = useTrafficStore((s) => s.current);
  const themeColor = useThemeStore((s) => s.color);
  const isDark = useThemeStore((s) => s.isDark);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rafId = window.requestAnimationFrame(() => {
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const dpr = window.devicePixelRatio || 1;
      const rect = canvas.getBoundingClientRect();
      const width = rect.width || 640;
      const height = rect.height || 210;

      if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
        canvas.width = width * dpr;
        canvas.height = height * dpr;
      }

      ctx.save();
      ctx.scale(dpr, dpr);
      ctx.clearRect(0, 0, width, height);

      const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';
      const secondaryHex = '#10b981'; // 上行互补翠绿色

      // 绘制网格参考线
      ctx.strokeStyle = isDark ? 'rgba(148, 163, 184, 0.12)' : 'rgba(100, 116, 139, 0.12)';
      ctx.lineWidth = 1;
      const rows = 4;
      for (let i = 1; i < rows; i++) {
        const y = (height / rows) * i;
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(width, y);
        ctx.stroke();
      }

      const maxVal = Math.max(
        64 * 1024,
        ...history.map((p) => Math.max(p.down, p.up))
      );

      const drawSmoothArea = (
        values: number[],
        strokeColor: string,
        fillTop: string,
        fillBottom: string
      ) => {
        if (values.length < 2) return;
        const stepX = width / (values.length - 1);
        const padTop = 18;
        const padBottom = 10;
        const usableH = height - padTop - padBottom;

        const pts = values.map((v, idx) => ({
          x: idx * stepX,
          y: padTop + usableH * (1 - Math.min(v / (maxVal * 1.15), 1)),
        }));

        ctx.beginPath();
        ctx.moveTo(pts[0].x, pts[0].y);
        for (let i = 0; i < pts.length - 1; i++) {
          const xc = (pts[i].x + pts[i + 1].x) / 2;
          const yc = (pts[i].y + pts[i + 1].y) / 2;
          ctx.quadraticCurveTo(pts[i].x, pts[i].y, xc, yc);
        }
        const last = pts[pts.length - 1];
        ctx.lineTo(last.x, last.y);

        ctx.strokeStyle = strokeColor;
        ctx.lineWidth = 2.4;
        ctx.stroke();

        // 渐变面积填充
        ctx.lineTo(width, height);
        ctx.lineTo(0, height);
        ctx.closePath();
        const grad = ctx.createLinearGradient(0, 0, 0, height);
        grad.addColorStop(0, fillTop);
        grad.addColorStop(1, fillBottom);
        ctx.fillStyle = grad;
        ctx.fill();
      };

      // 先画下行曲线（主题色），再画上行曲线（互补绿）
      drawSmoothArea(
        history.map((p) => p.down),
        primaryHex,
        primaryHex + '55',
        primaryHex + '05'
      );
      drawSmoothArea(
        history.map((p) => p.up),
        secondaryHex,
        secondaryHex + '44',
        secondaryHex + '05'
      );

      ctx.restore();
    });

    return () => window.cancelAnimationFrame(rafId);
  }, [history, themeColor, isDark]);

  const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 10,
          fontSize: 13,
        }}
      >
        <span style={{ fontWeight: 600, opacity: 0.85 }}>
          实时上下行流量波形图 (最近 60 秒)
        </span>
        <div style={{ display: 'flex', gap: 18 }}>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: primaryHex,
                display: 'inline-block',
              }}
            />
            下行：{formatSpeed(current.downloadSpeed)}
          </span>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: '#10b981',
                display: 'inline-block',
              }}
            />
            上行：{formatSpeed(current.uploadSpeed)}
          </span>
        </div>
      </div>

      <canvas
        ref={canvasRef}
        style={{
          width: '100%',
          height: 205,
          display: 'block',
          borderRadius: 10,
          background: isDark ? 'rgba(15, 23, 42, 0.5)' : 'rgba(241, 245, 249, 0.65)',
        }}
      />
    </div>
  );
};
