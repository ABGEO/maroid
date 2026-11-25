BEGIN;

CREATE VIEW log_entry_instalments WITH (security_invoker = true) AS
SELECT maintenance_id AS log_entry_id, number, due_on, amount
FROM maintenance_instalments
UNION ALL
SELECT expense_id, number, due_on, amount
FROM expense_instalments
UNION ALL
SELECT income_id, number, due_on, amount
FROM income_instalments;

COMMENT ON VIEW log_entry_instalments IS
    'The instalments of each maintenance, each expense, and each income.';

COMMIT;
