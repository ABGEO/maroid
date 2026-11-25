BEGIN;

CREATE VIEW log_entry_lines WITH (security_invoker = true) AS
SELECT f.refuel_id AS log_entry_id, f.position, f.fuel_id AS item_id, fu.name AS item,
       f.amount, f.unit_price, f.volume, f.full_tank, f.free
FROM refuel_fuels f
LEFT JOIN fuels fu ON fu.id = f.fuel_id
UNION ALL
SELECT l.maintenance_id, l.position, l.maintenance_type_id, mt.name,
       l.amount, NULL, NULL, NULL, NULL
FROM maintenance_lines l
LEFT JOIN maintenance_types mt ON mt.id = l.maintenance_type_id
UNION ALL
SELECT l.expense_id, l.position, l.expense_type_id, et.name,
       l.amount, NULL, NULL, NULL, NULL
FROM expense_lines l
LEFT JOIN expense_types et ON et.id = l.expense_type_id;

COMMENT ON VIEW log_entry_lines IS
    'The fuels of each refuel, and the types of each maintenance and each expense, with the name of each item.';

COMMIT;
