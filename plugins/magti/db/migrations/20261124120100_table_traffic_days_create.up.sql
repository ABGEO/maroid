BEGIN;

CREATE TABLE traffic_days
(
    id                UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    subscriber_number TEXT        NOT NULL CHECK (char_length(subscriber_number) BETWEEN 1 AND 64),
    day               DATE        NOT NULL,
    data_mb           NUMERIC,
    call_in_seconds   INTEGER,
    call_out_seconds  INTEGER,
    sms_received      INTEGER,
    sms_sent          INTEGER,
    collected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The portal corrects the traffic of a day, so a day holds one row that takes the
    -- latest values.
    CONSTRAINT traffic_days_identity_key UNIQUE (workspace_id, subscriber_number, day)
);

COMMENT ON TABLE traffic_days IS
    'Scoped to a workspace: the traffic of a mobile subscriber on a day of Tbilisi. A day outlives its assignment.';

ALTER TABLE traffic_days
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE traffic_days
    FORCE ROW LEVEL SECURITY;

-- MAGTIACC-INV-002: The traffic of one workspace serves no other workspace.
CREATE POLICY traffic_days_workspace_isolation ON traffic_days
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
