CREATE TABLE IF NOT EXISTS rollouts (
    id text PRIMARY KEY,
    service text NOT NULL,
    cluster text NOT NULL,
    namespace text NOT NULL,
    stable_version text NOT NULL,
    candidate_version text NOT NULL,
    candidate_image text NOT NULL,
    state text NOT NULL,
    weight integer NOT NULL,
    analysis text NOT NULL DEFAULT '',
    proposal_action text NOT NULL DEFAULT '',
    ai_action text NOT NULL DEFAULT '',
    ai_summary text NOT NULL DEFAULT '',
    ai_status text NOT NULL DEFAULT '',
    ai_mismatch boolean NOT NULL DEFAULT false,
    verification text NOT NULL DEFAULT '',
    requests double precision NOT NULL DEFAULT 0,
    error_rate double precision NOT NULL DEFAULT 0,
    p95 double precision NOT NULL DEFAULT 0,
    stable_error_rate double precision NOT NULL DEFAULT 0,
    stable_p95 double precision NOT NULL DEFAULT 0,
    candidate_ready boolean NOT NULL DEFAULT false,
    stage_started_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS rollouts_one_payment_canary
    ON rollouts (namespace, service)
    WHERE state IN ('PENDING', 'RUNNING', 'ANALYZING', 'AWAITING_APPROVAL', 'PROMOTING', 'ROLLING_BACK');

CREATE TABLE IF NOT EXISTS rollout_stages (
    id text PRIMARY KEY,
    rollout_id text NOT NULL REFERENCES rollouts (id),
    weight integer NOT NULL,
    started_at timestamptz NOT NULL,
    ended_at timestamptz
);

CREATE TABLE IF NOT EXISTS rollout_analysis (
    id text PRIMARY KEY,
    rollout_id text NOT NULL REFERENCES rollouts (id),
    weight integer NOT NULL,
    result text NOT NULL,
    requests double precision NOT NULL DEFAULT 0,
    error_rate double precision NOT NULL DEFAULT 0,
    p95 double precision NOT NULL DEFAULT 0,
    ready boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rollout_events (
    id text PRIMARY KEY,
    rollout_id text NOT NULL REFERENCES rollouts (id),
    at timestamptz NOT NULL,
    title text NOT NULL,
    detail text NOT NULL DEFAULT '',
    kind text NOT NULL
);
