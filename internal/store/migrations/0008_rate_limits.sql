-- Persistent, restart-surviving rate-limit state, shared by the gRPC
-- interceptor and HTTP middleware. One row per bucket key; the key is a
-- composite of the tier (per-ip / per-account / per-project) and the
-- identifier it limits. Fixed-window counting: window_start is the aligned
-- start of the current window (RFC3339Nano UTC), count the hits inside it.
CREATE TABLE rate_limits (
    key          TEXT PRIMARY KEY,
    count        INTEGER NOT NULL,
    window_start TEXT NOT NULL
);

CREATE INDEX idx_rate_limits_window_start ON rate_limits (window_start);
