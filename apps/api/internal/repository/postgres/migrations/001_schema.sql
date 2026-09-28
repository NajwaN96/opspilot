CREATE TABLE clusters (
    id text PRIMARY KEY,
    name text NOT NULL,
    environment text NOT NULL,
    region text NOT NULL,
    kubernetes_version text NOT NULL DEFAULT '',
    status text NOT NULL,
    provider text NOT NULL,
    simulated boolean NOT NULL,
    source text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE services (
    id text PRIMARY KEY,
    cluster_id text NOT NULL REFERENCES clusters (id),
    name text NOT NULL,
    namespace text NOT NULL,
    status text NOT NULL,
    owner text NOT NULL DEFAULT '',
    runtime text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    version text NOT NULL DEFAULT '',
    previous_version text NOT NULL DEFAULT '',
    source text NOT NULL,
    telemetry text NOT NULL DEFAULT 'none',
    image text NOT NULL DEFAULT '',
    workload_kind text NOT NULL DEFAULT '',
    desired_replicas integer NOT NULL DEFAULT 0,
    ready_replicas integer NOT NULL DEFAULT 0,
    restarts integer NOT NULL DEFAULT 0,
    availability double precision NOT NULL DEFAULT 0,
    p95_latency_ms integer NOT NULL DEFAULT 0,
    error_rate double precision NOT NULL DEFAULT 0,
    last_observed_at timestamptz,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX services_cluster_source_idx ON services (cluster_id, source);
CREATE INDEX services_namespace_idx ON services (namespace);

CREATE TABLE incidents (
    id text PRIMARY KEY,
    severity text NOT NULL,
    service_id text NOT NULL,
    service_name text NOT NULL,
    title text NOT NULL,
    status text NOT NULL,
    started_at timestamptz NOT NULL,
    resolved_at timestamptz,
    summary text NOT NULL DEFAULT '',
    cluster_name text NOT NULL,
    source text NOT NULL DEFAULT 'simulation',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX incidents_status_idx ON incidents (status);

CREATE TABLE incident_events (
    id bigserial PRIMARY KEY,
    incident_id text NOT NULL REFERENCES incidents (id),
    at timestamptz NOT NULL,
    clock text NOT NULL DEFAULT '',
    title text NOT NULL,
    detail text NOT NULL DEFAULT '',
    kind text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX incident_events_incident_idx ON incident_events (incident_id, at);

CREATE TABLE remediation_proposals (
    id text PRIMARY KEY,
    incident_id text NOT NULL REFERENCES incidents (id),
    action text NOT NULL,
    from_version text NOT NULL,
    to_version text NOT NULL,
    risk text NOT NULL DEFAULT '',
    policy_result text NOT NULL,
    status text NOT NULL,
    simulated boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX remediation_proposals_incident_idx ON remediation_proposals (incident_id);

CREATE TABLE approvals (
    id text PRIMARY KEY,
    proposal_id text NOT NULL REFERENCES remediation_proposals (id),
    incident_id text NOT NULL REFERENCES incidents (id),
    actor text NOT NULL,
    decision text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX approvals_incident_idx ON approvals (incident_id, created_at);

CREATE TABLE remediation_executions (
    id text PRIMARY KEY,
    proposal_id text NOT NULL REFERENCES remediation_proposals (id),
    incident_id text NOT NULL REFERENCES incidents (id),
    status text NOT NULL,
    simulated boolean NOT NULL DEFAULT true,
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    verification_result text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX remediation_executions_incident_idx ON remediation_executions (incident_id);

CREATE TABLE audit_events (
    id bigserial PRIMARY KEY,
    at timestamptz NOT NULL,
    actor text NOT NULL,
    incident_id text,
    proposal_id text,
    action text NOT NULL,
    policy_result text NOT NULL DEFAULT '',
    approval_result text NOT NULL DEFAULT '',
    execution_status text NOT NULL DEFAULT '',
    verification_result text NOT NULL DEFAULT '',
    detail text NOT NULL DEFAULT ''
);

CREATE INDEX audit_events_incident_idx ON audit_events (incident_id, at);

CREATE TABLE kubernetes_resources (
    id text PRIMARY KEY,
    cluster_id text NOT NULL,
    namespace text NOT NULL,
    kind text NOT NULL,
    name text NOT NULL,
    version text NOT NULL DEFAULT '',
    image text NOT NULL DEFAULT '',
    desired_replicas integer NOT NULL DEFAULT 0,
    ready_replicas integer NOT NULL DEFAULT 0,
    restarts integer NOT NULL DEFAULT 0,
    status text NOT NULL,
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_observed_at timestamptz NOT NULL,
    UNIQUE (cluster_id, namespace, kind, name)
);

CREATE INDEX kubernetes_resources_namespace_idx ON kubernetes_resources (namespace, kind);

CREATE TABLE kubernetes_events (
    id text PRIMARY KEY,
    namespace text NOT NULL,
    event_type text NOT NULL,
    reason text NOT NULL,
    object_ref text NOT NULL,
    message text NOT NULL DEFAULT '',
    count integer NOT NULL DEFAULT 1,
    at timestamptz NOT NULL
);

CREATE INDEX kubernetes_events_namespace_idx ON kubernetes_events (namespace, at DESC);

CREATE TABLE demo_state (
    id integer PRIMARY KEY CHECK (id = 1),
    epoch timestamptz NOT NULL,
    remediation_started_at timestamptz
);
