import { useEffect, useRef, useState, type ReactNode } from "react";

/*
 * Hand-rolled SVG charts — no chart library, no external requests. Data ink
 * is the single `--chart-series` hue (validated against both surfaces);
 * chrome uses border/text tokens; functional colors never appear as series.
 * Numbers render with tabular figures so live values do not jitter.
 */

// useWidth measures the rendered width of a wrapper so SVGs draw at native
// pixel scale (no preserveAspectRatio distortion of stroke widths).
function useWidth(): [React.RefObject<HTMLDivElement | null>, number] {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width ?? 0;
      setWidth(Math.floor(w));
    });
    ro.observe(el);
    setWidth(Math.floor(el.getBoundingClientRect().width));
    return () => ro.disconnect();
  }, []);
  return [ref, width];
}

// niceStep rounds up to 1/2/5 × 10^n so axis ticks are round integers.
function niceStep(raw: number): number {
  if (raw <= 1) return 1;
  const pow = Math.pow(10, Math.floor(Math.log10(raw)));
  const frac = raw / pow;
  const nice = frac <= 1 ? 1 : frac <= 2 ? 2 : frac <= 5 ? 5 : 10;
  return nice * pow;
}

const TICKS = 3; // horizontal gridlines above the baseline

// ---------- StatTile ----------

// StatTile is a headline number, not a chart: value, label, optional delta
// vs the previous period and a caption hint. `title` becomes the browser
// tooltip (used to document the DAU approximation).
export function StatTile({
  label,
  value,
  delta,
  hint,
  title,
  warning,
}: {
  label: string;
  value: string;
  // Current vs previous period; renders "↑ 12% vs previous" tinted by
  // direction (up = success). Omitted when previous is zero (no baseline).
  delta?: { current: number; previous: number };
  hint?: string;
  title?: string;
  warning?: boolean;
}) {
  let deltaNode: ReactNode = null;
  if (delta && delta.previous > 0 && delta.current !== delta.previous) {
    const change = (delta.current - delta.previous) / delta.previous;
    const up = change > 0;
    deltaNode = (
      <span className={`stat-tile__delta ${up ? "text-success" : "text-danger"}`}>
        {up ? "↑" : "↓"} {Math.abs(Math.round(change * 100))}% vs previous
      </span>
    );
  }
  return (
    <div className={`card stat-tile${warning ? " stat-tile--warning" : ""}`} title={title}>
      <span className="caption">{label}</span>
      <span className="stat-tile__value">{value}</span>
      {deltaNode}
      {hint ? <span className="caption text-tertiary">{hint}</span> : null}
    </div>
  );
}

// ---------- Sparkline ----------

// Sparkline is a tiny, axis-free trend line for cards. Decorative summary
// only — it carries no labels and is hidden from assistive tech.
export function Sparkline({
  values,
  width = 120,
  height = 28,
}: {
  values: number[];
  width?: number;
  height?: number;
}) {
  if (values.length === 0) return null;
  const max = Math.max(...values, 1);
  const pad = 2;
  const innerW = width - pad * 2;
  const innerH = height - pad * 2;
  const x = (i: number) => pad + (values.length === 1 ? innerW / 2 : (i * innerW) / (values.length - 1));
  const y = (v: number) => pad + innerH - (v / max) * innerH;
  const d = values.map((v, i) => `${i === 0 ? "M" : "L"}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(" ");
  return (
    <svg width={width} height={height} aria-hidden="true">
      <path
        d={d}
        fill="none"
        stroke="var(--chart-series)"
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

// ---------- LineChart ----------

export type LineSeries = {
  label: string;
  values: number[];
  // Defaults to the chart series hue; multi-series charts must pass fixed
  // per-entity colors (identity never cycles or reassigns).
  color?: string;
};

// LineChart draws one x axis of calendar days and one y axis (never dual).
// Hover shows a crosshair with a shared-x tooltip listing every series.
// With ≥2 series a legend renders; a single series is named by the card
// title instead.
export function LineChart({
  labels,
  series,
  height = 160,
  formatValue = (v) => String(v),
}: {
  labels: string[];
  series: LineSeries[];
  height?: number;
  formatValue?: (v: number) => string;
}) {
  const [ref, width] = useWidth();
  const [hover, setHover] = useState<number | null>(null);

  const n = labels.length;
  const maxValue = Math.max(1, ...series.flatMap((s) => s.values));
  const step = niceStep(Math.ceil(maxValue / TICKS));
  const yMax = step * TICKS;

  const padLeft = 8 + String(formatValue(yMax)).length * 8;
  const pad = { top: 8, right: 8, bottom: 20, left: padLeft };
  const innerW = Math.max(0, width - pad.left - pad.right);
  const innerH = height - pad.top - pad.bottom;
  const x = (i: number) => pad.left + (n <= 1 ? innerW / 2 : (i * innerW) / (n - 1));
  // Clamp to the plot band so a net-negative point (e.g. a refund-heavy revenue
  // month) sits on the baseline instead of drawing below the axis / off-canvas
  // and breaking the area path and hover circle. The tooltip still shows the
  // true (possibly negative) value.
  const y = (v: number) => {
    const raw = pad.top + innerH - (v / yMax) * innerH;
    return Math.max(pad.top, Math.min(pad.top + innerH, raw));
  };

  function onMove(e: React.MouseEvent<SVGSVGElement>) {
    if (n === 0 || innerW <= 0) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const px = e.clientX - rect.left - pad.left;
    const i = Math.round((px / innerW) * (n - 1));
    setHover(Math.max(0, Math.min(n - 1, i)));
  }

  const ticks = Array.from({ length: TICKS + 1 }, (_, i) => i * step);
  // First / middle / last day labels keep the x axis recessive.
  const xTicks = n <= 2 ? labels.map((_, i) => i) : [0, Math.floor((n - 1) / 2), n - 1];
  const shortDay = (d: string) => d.slice(5);

  const tipOnRight = hover !== null && n > 1 && hover / (n - 1) < 0.6;

  return (
    <div className="stack-12">
      {series.length >= 2 && (
        <div className="row-16">
          {series.map((s) => (
            <span key={s.label} className="row-8 caption">
              <svg width="12" height="12" aria-hidden="true">
                <rect width="12" height="4" y="4" rx="2" fill={s.color ?? "var(--chart-series)"} />
              </svg>
              {s.label}
            </span>
          ))}
        </div>
      )}
      <div className="chart" ref={ref}>
        {width > 0 && (
          <svg
            width={width}
            height={height}
            role="img"
            onMouseMove={onMove}
            onMouseLeave={() => setHover(null)}
          >
            {/* Recessive grid: hairlines at each tick, no frame. */}
            {ticks.map((t) => (
              <g key={t}>
                <line
                  x1={pad.left}
                  x2={width - pad.right}
                  y1={y(t)}
                  y2={y(t)}
                  stroke="var(--border)"
                  strokeWidth={1}
                />
                <text
                  x={pad.left - 6}
                  y={y(t) + 3}
                  textAnchor="end"
                  fontSize={10}
                  fontFamily="var(--font-mono)"
                  fill="var(--text-tertiary)"
                >
                  {formatValue(t)}
                </text>
              </g>
            ))}
            {xTicks.map((i) => (
              <text
                key={i}
                x={x(i)}
                y={height - 4}
                textAnchor={i === 0 ? "start" : i === n - 1 ? "end" : "middle"}
                fontSize={10}
                fontFamily="var(--font-mono)"
                fill="var(--text-tertiary)"
              >
                {shortDay(labels[i])}
              </text>
            ))}
            {hover !== null && (
              <line
                x1={x(hover)}
                x2={x(hover)}
                y1={pad.top}
                y2={pad.top + innerH}
                stroke="var(--border-strong)"
                strokeWidth={1}
              />
            )}
            {series.map((s) => {
              const color = s.color ?? "var(--chart-series)";
              const d = s.values
                .map((v, i) => `${i === 0 ? "M" : "L"}${x(i).toFixed(1)},${y(v).toFixed(1)}`)
                .join(" ");
              const area =
                n > 1
                  ? `${d} L${x(n - 1).toFixed(1)},${y(0)} L${x(0).toFixed(1)},${y(0)} Z`
                  : "";
              return (
                <g key={s.label}>
                  {area && (
                    <path
                      d={area}
                      fill={color}
                      opacity={0.07}
                      stroke="none"
                    />
                  )}
                  <path
                    d={d}
                    fill="none"
                    stroke={color}
                    strokeWidth={2}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                  {hover !== null && s.values[hover] !== undefined && (
                    <circle
                      cx={x(hover)}
                      cy={y(s.values[hover])}
                      r={3.5}
                      fill="var(--surface)"
                      stroke={color}
                      strokeWidth={2}
                    />
                  )}
                </g>
              );
            })}
          </svg>
        )}
        {hover !== null && labels[hover] !== undefined && (
          <div
            className="chart-tip"
            style={
              tipOnRight
                ? { left: x(hover) + 12 }
                : { left: x(hover) - 12, transform: "translateX(-100%)" }
            }
          >
            <div className="stack-8" style={{ gap: 2 }}>
              <span className="text-secondary mono">{labels[hover]}</span>
              {series.map((s) => (
                <span key={s.label} className="row-8">
                  <span className="text-secondary">{s.label}</span>
                  <span className="chart-tip__value">
                    {formatValue(s.values[hover] ?? 0)}
                  </span>
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ---------- BarBreakdown ----------

// BarBreakdown renders share-of-total as direct-labeled horizontal bars.
// Identity is carried by position and label (fixed order per entity), so
// the fills stay on the single chart hue — no categorical palette needed.
export function BarBreakdown({ items }: { items: { label: string; value: number }[] }) {
  // Share is of total magnitude (sum of absolute values), so a net-negative
  // entry — a refund-heavy store/tier revenue — does not blow the track up:
  // its share stays a sane percentage and the fill width is clamped to
  // [0, 100] rather than going CSS-invalid negative or overflowing past 100%.
  // For the all-non-negative auth breakdowns this is identical to a plain sum.
  const total = items.reduce((sum, it) => sum + Math.abs(it.value), 0);
  return (
    <div className="bars">
      {items.map((it) => {
        const pct = total > 0 ? (it.value / total) * 100 : 0;
        const width = Math.max(0, Math.min(100, pct));
        return (
          <div
            key={it.label}
            className="bars__row"
            title={`${it.label}: ${it.value} (${Math.round(pct)}%)`}
          >
            <span className="bars__label">{it.label}</span>
            <div className="bars__track">
              <div className="bars__fill" style={{ width: `${width}%` }} />
            </div>
            <span className="bars__value">
              {it.value} · {Math.round(pct)}%
            </span>
          </div>
        );
      })}
    </div>
  );
}
