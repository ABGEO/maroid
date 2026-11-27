BEGIN;

CREATE TABLE contributions
(
    id                UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id           UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    basis_id          UUID        NOT NULL,
    source            TEXT        NOT NULL,
    contribution_type TEXT        NOT NULL,
    paid_on           DATE        NOT NULL,
    closed_on         DATE,
    salary_year       SMALLINT,
    salary_month      SMALLINT CHECK (salary_month BETWEEN 1 AND 12),
    gross_salary      NUMERIC     NOT NULL,
    amount            NUMERIC,
    units             NUMERIC,
    employer_code     TEXT REFERENCES employers (code) ON DELETE RESTRICT,
    collected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One basis pays once from each source, so the amount stays out of the identity.
    CONSTRAINT contributions_identity_key UNIQUE (user_id, basis_id, source)
);

COMMENT ON TABLE contributions IS
    'Scoped to a user: the contributions to the pension account of one person, in every workspace.';

CREATE INDEX contributions_user_paid_on_idx
    ON contributions (user_id, paid_on DESC, id DESC);
CREATE INDEX contributions_user_employer_paid_on_idx
    ON contributions (user_id, employer_code, paid_on DESC, id DESC);

ALTER TABLE contributions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE contributions
    FORCE ROW LEVEL SECURITY;

-- PENSION-INV-002: A row of one user serves no other user, in any workspace.
CREATE POLICY contributions_user_isolation ON contributions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
