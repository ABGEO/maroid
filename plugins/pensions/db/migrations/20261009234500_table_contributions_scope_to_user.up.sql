BEGIN;

COMMENT ON TABLE organizations IS
    'Shared: the employers that the pension agency names, the same for every person.';

-- The rows predate the user and belong to none, so the table starts empty.
DROP TABLE contributions;

CREATE TABLE contributions
(
    id                UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id           UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    hash              TEXT        NOT NULL,
    basis_id          UUID        NOT NULL,
    date              TIMESTAMPTZ NOT NULL,
    closing_date      TIMESTAMPTZ NULL,
    year              SMALLINT    NULL,
    month             SMALLINT    NULL,
    gross_salary      DECIMAL     NOT NULL,
    type              TEXT        NOT NULL,
    source            TEXT        NOT NULL,
    amount            DECIMAL     NULL,
    units             DECIMAL     NULL,
    organization_code TEXT        NULL REFERENCES organizations (code) ON DELETE RESTRICT,
    CONSTRAINT contributions_user_hash_key UNIQUE (user_id, hash)
);

COMMENT ON TABLE contributions IS
    'Scoped to a user: the contributions of one person, in every workspace.';

ALTER TABLE contributions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE contributions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY contributions_user_isolation ON contributions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
