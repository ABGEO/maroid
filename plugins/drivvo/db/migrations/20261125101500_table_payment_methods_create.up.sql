BEGIN;

CREATE TABLE payment_methods
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT payment_methods_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE payment_methods IS
    'Scoped to a workspace: the payment methods of the car log account.';

ALTER TABLE payment_methods
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_methods
    FORCE ROW LEVEL SECURITY;

CREATE POLICY payment_methods_workspace_isolation ON payment_methods
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
