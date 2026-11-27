BEGIN;

CREATE TABLE portfolio_shares
(
    id                    UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id               UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    portfolio_id          BIGINT      NOT NULL,
    portfolio_name_ka     TEXT        NOT NULL,
    portfolio_name_en     TEXT        NOT NULL,
    asset_manager_name_ka TEXT        NOT NULL,
    asset_manager_name_en TEXT        NOT NULL,
    ticker                TEXT        NOT NULL,
    risk_tier_name_ka     TEXT        NOT NULL,
    risk_tier_name_en     TEXT        NOT NULL,
    target_share          NUMERIC     NOT NULL,
    current_share         NUMERIC     NOT NULL,
    amount                NUMERIC     NOT NULL,
    units                 NUMERIC     NOT NULL,
    unit_price            NUMERIC     NOT NULL,
    collected_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT portfolio_shares_user_portfolio_key UNIQUE (user_id, portfolio_id)
);

COMMENT ON TABLE portfolio_shares IS
    'Scoped to a user: the current share of each portfolio in the pension account of one person.';

ALTER TABLE portfolio_shares
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE portfolio_shares
    FORCE ROW LEVEL SECURITY;

-- PENSION-INV-002: A row of one user serves no other user, in any workspace.
CREATE POLICY portfolio_shares_user_isolation ON portfolio_shares
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
