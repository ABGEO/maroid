BEGIN;

CREATE TABLE account_summaries
(
    id                     UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id                UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    asset_value            NUMERIC     NOT NULL,
    units                  NUMERIC     NOT NULL,
    gain                   NUMERIC     NOT NULL,
    withheld_amount        NUMERIC     NOT NULL,
    person_contributions   NUMERIC     NOT NULL,
    employer_contributions NUMERIC     NOT NULL,
    state_contributions    NUMERIC     NOT NULL,
    total_contributions    NUMERIC     NOT NULL,
    collected_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT account_summaries_user_key UNIQUE (user_id)
);

COMMENT ON TABLE account_summaries IS
    'Scoped to a user: the latest summary of the pension account of one person.';

ALTER TABLE account_summaries
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE account_summaries
    FORCE ROW LEVEL SECURITY;

-- PENSION-INV-002: A row of one user serves no other user, in any workspace.
CREATE POLICY account_summaries_user_isolation ON account_summaries
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
