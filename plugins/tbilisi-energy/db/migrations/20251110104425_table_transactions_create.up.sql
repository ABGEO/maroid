BEGIN;

CREATE TABLE IF NOT EXISTS transactions
(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4()
);

COMMIT;
