ALTER TABLE incidents ADD COLUMN IF NOT EXISTS fingerprint text NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS rule_id text NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS telemetry_recovered_at timestamptz;

CREATE UNIQUE INDEX IF NOT EXISTS incidents_one_active_fingerprint
    ON incidents (fingerprint)
    WHERE fingerprint <> '' AND status <> 'resolved';

CREATE TABLE IF NOT EXISTS lab_experiments (
    id text PRIMARY KEY,
    actor text NOT NULL,
    service_id text NOT NULL,
    service_name text NOT NULL,
    scenario text NOT NULL,
    scenario_name text NOT NULL DEFAULT '',
    cluster_name text NOT NULL,
    namespace text NOT NULL,
    status text NOT NULL,
    simulated boolean NOT NULL DEFAULT false,
    duration_sec integer NOT NULL,
    parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
    note text NOT NULL DEFAULT '',
    started_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    stopped_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS lab_experiments_status_idx ON lab_experiments (status, expires_at);
