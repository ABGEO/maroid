BEGIN;

CREATE TABLE asset_values
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    valued_on    DATE        NOT NULL,
    asset_value  NUMERIC     NOT NULL,
    units        NUMERIC     NOT NULL,
    unit_price   NUMERIC     NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT asset_values_user_valued_on_key UNIQUE (user_id, valued_on)
);

COMMENT ON TABLE asset_values IS
    'Scoped to a user: the asset value of the pension account of one person on each date that the pension service reports.';

ALTER TABLE asset_values
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_values
    FORCE ROW LEVEL SECURITY;

-- PENSION-INV-002: A row of one user serves no other user, in any workspace.
CREATE POLICY asset_values_user_isolation ON asset_values
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
