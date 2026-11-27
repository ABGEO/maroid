BEGIN;

CREATE TABLE portfolio_unit_prices
(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    portfolio_id    BIGINT      NOT NULL,
    priced_on       DATE        NOT NULL,
    unit_price      NUMERIC     NOT NULL,
    net_asset_value NUMERIC,
    units           NUMERIC,
    collected_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT portfolio_unit_prices_portfolio_priced_on_key UNIQUE (portfolio_id, priced_on)
);

COMMENT ON TABLE portfolio_unit_prices IS
    'Shared: the unit price of each portfolio on each day. A unit price names no person, and two people of one portfolio read the same row.';

COMMIT;
