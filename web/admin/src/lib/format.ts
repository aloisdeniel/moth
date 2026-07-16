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
