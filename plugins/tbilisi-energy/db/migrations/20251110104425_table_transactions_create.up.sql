BEGIN;

CREATE TABLE IF NOT EXISTS dev_maroid_tbilisi_energy.Transactions
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4()
);

COMMIT;
