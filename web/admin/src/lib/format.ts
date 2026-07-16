import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";

// formatDate renders a proto timestamp as "2026-07-16" (mono, data-like
// per the design system).
export function formatDate(ts: Timestamp | undefined): string {
  if (!ts) return "—";
  return timestampDate(ts).toISOString().slice(0, 10);
}

// formatDateTime renders "2026-07-16 14:03".
export function formatDateTime(ts: Timestamp | undefined): string {
  if (!ts) return "—";
  const iso = timestampDate(ts).toISOString();
  return iso.slice(0, 10) + " " + iso.slice(11, 16);
}

// formatRelative renders "just now" / "5m ago" / "3h ago" / "12d ago",
// falling back to the date beyond a month (activity feeds).
export function formatRelative(ts: Timestamp | undefined): string {
  if (!ts) return "—";
  const s = Math.max(0, Math.floor((Date.now() - timestampDate(ts).getTime()) / 1000));
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 31) return `${d}d ago`;
  return formatDate(ts);
}

// localDay renders a Date as "YYYY-MM-DD" in the browser's timezone — the
// closest client-side stand-in for the project's rollup timezone when
// composing analytics date ranges.
export function localDay(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

// dayAgo returns the local "YYYY-MM-DD" n days before today (0 = today).
export function dayAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return localDay(d);
}
