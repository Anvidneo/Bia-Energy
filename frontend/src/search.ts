// Extracts a meter id from a free-text search query. The topbar search box
// only understands meter ids for now (e.g. "M-110", "110", "m-110") — it
// pulls the first run of digits out of the query and normalizes it to the
// dataset's "M-<digits>" id format. Anomaly/full-text search is out of
// scope for this pass.
export function extractMeterIdFromQuery(query: string): string | null {
  const digits = query.match(/\d+/)?.[0]
  return digits ? `M-${digits}` : null
}
