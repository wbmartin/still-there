// Epoch-second timestamps (every *Ts the API sends) as people read them.

/** "never" when absent, else how long ago: "just now", "5 min ago", ... */
export const formatAgo = (
  ts: number | undefined,
  nowMs: number = Date.now(),
): string => {
  if (ts === undefined) return "never";
  const agoSec = Math.max(0, Math.floor(nowMs / 1000 - ts));
  if (agoSec < 60) return "just now";
  if (agoSec < 3600) return `${Math.floor(agoSec / 60)} min ago`;
  if (agoSec < 86400) return `${Math.floor(agoSec / 3600)} h ago`;
  return `${Math.floor(agoSec / 86400)} d ago`;
};

/** The local date and time, or "never" when absent. */
export const formatTs = (ts: number | undefined): string =>
  ts === undefined ? "never" : new Date(ts * 1000).toLocaleString();
