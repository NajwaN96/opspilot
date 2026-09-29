CREATE TABLE IF NOT EXISTS incident_evidence_snapshots (
    id text PRIMARY KEY,
    incident_id text NOT NULL REFERENCES incidents (id),
    version integer NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (incident_id, version)
);

CREATE TABLE IF NOT EXISTS ai_investigations (
    id text PRIMARY KEY,
    incident_id text NOT NULL REFERENCES incidents (id),
    snapshot_id text NOT NULL,
    snapshot_version integer NOT NULL,
    provider text NOT NULL,
    model text NOT NULL DEFAULT '',
    status text NOT NULL,
    output jsonb NOT NULL DEFAULT '{}'::jsonb,
    validation jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_category text NOT NULL DEFAULT '',
    prompt_tokens integer NOT NULL DEFAULT 0,
    completion_tokens integer NOT NULL DEFAULT 0,
    real boolean NOT NULL DEFAULT false,
    started_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ai_investigations_one_active
    ON ai_investigations (incident_id, snapshot_version)
    WHERE status IN ('queued', 'collecting_evidence', 'investigating', 'validating');
