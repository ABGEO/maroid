BEGIN;

CREATE TABLE car_log_settings
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    volume_unit        TEXT        NOT NULL CHECK (volume_unit IN ('l', 'gal_us', 'gal_imp')),
    consumption_format SMALLINT    NOT NULL CHECK (consumption_format BETWEEN 1 AND 12),
    currency           TEXT        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT car_log_settings_workspace_key UNIQUE (workspace_id)
);

COMMENT ON TABLE car_log_settings IS
    'Scoped to a workspace: the units and the currency of the car log account.';

ALTER TABLE car_log_settings
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE car_log_settings
    FORCE ROW LEVEL SECURITY;

CREATE POLICY car_log_settings_workspace_isolation ON car_log_settings
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
