-- Bia Energy schema. CREATE TABLE IF NOT EXISTS is enough for this MVP —
-- no migration tool, applied once at startup (see db.go).

CREATE TABLE IF NOT EXISTS meters (
    id TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS readings (
    id BIGSERIAL PRIMARY KEY,
    meter_id TEXT NOT NULL REFERENCES meters(id),
    ts TIMESTAMPTZ NOT NULL,
    consumption_kwh DOUBLE PRECISION NOT NULL,
    voltage_v DOUBLE PRECISION NOT NULL,
    current_a DOUBLE PRECISION NOT NULL,
    power_factor DOUBLE PRECISION NOT NULL,
    UNIQUE (meter_id, ts)
);

CREATE INDEX IF NOT EXISTS idx_readings_meter_ts ON readings (meter_id, ts);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    meter_id TEXT NOT NULL REFERENCES meters(id),
    event_ts TIMESTAMPTZ NOT NULL,
    event_type TEXT NOT NULL,
    description TEXT NOT NULL,
    UNIQUE (meter_id, event_ts, event_type)
);

CREATE TABLE IF NOT EXISTS anomalies (
    id BIGSERIAL PRIMARY KEY,
    meter_id TEXT NOT NULL REFERENCES meters(id),
    detected_at TIMESTAMPTZ NOT NULL,
    type TEXT NOT NULL,
    severity TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    reason TEXT NOT NULL,
    recommended_action TEXT NOT NULL,
    related_event_type TEXT,
    related_event_at TIMESTAMPTZ,
    evidence_json JSONB NOT NULL,
    analysis_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_anomalies_meter ON anomalies (meter_id);
CREATE INDEX IF NOT EXISTS idx_anomalies_analysis ON anomalies (analysis_id);
