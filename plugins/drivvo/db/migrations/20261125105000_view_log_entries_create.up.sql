BEGIN;

CREATE VIEW log_entries WITH (security_invoker = true) AS
SELECT r.id, r.vehicle_id, 'refuel'::text AS kind, r.occurred_at, r.odometer, r.amount,
       NULL::text AS title, r.note, st.name AS station, NULL::text AS place,
       pm.name AS payment_method, re.name AS reason, NULL::text AS income_type,
       r.missed_refuel, r.battery_start_pct, r.battery_end_pct, r.charge_minutes
FROM refuels r
LEFT JOIN stations st ON st.id = r.station_id
LEFT JOIN payment_methods pm ON pm.id = r.payment_method_id
LEFT JOIN reasons re ON re.id = r.reason_id
UNION ALL
SELECT m.id, m.vehicle_id, 'maintenance', m.occurred_at, m.odometer, m.amount,
       m.title, m.note, NULL, pl.name,
       pm.name, NULL, NULL,
       false, NULL, NULL, NULL
FROM maintenances m
LEFT JOIN places pl ON pl.id = m.place_id
LEFT JOIN payment_methods pm ON pm.id = m.payment_method_id
UNION ALL
SELECT e.id, e.vehicle_id, 'expense', e.occurred_at, e.odometer, e.amount,
       e.title, e.note, NULL, pl.name,
       pm.name, re.name, NULL,
       false, NULL, NULL, NULL
FROM expenses e
LEFT JOIN places pl ON pl.id = e.place_id
LEFT JOIN payment_methods pm ON pm.id = e.payment_method_id
LEFT JOIN reasons re ON re.id = e.reason_id
UNION ALL
SELECT i.id, i.vehicle_id, 'income', i.occurred_at, i.odometer, i.amount,
       NULL, i.note, NULL, NULL,
       NULL, NULL, it.name,
       false, NULL, NULL, NULL
FROM incomes i
LEFT JOIN income_types it ON it.id = i.income_type_id
UNION ALL
SELECT o.id, o.vehicle_id, 'odometer', o.occurred_at, o.odometer, NULL,
       NULL, o.note, NULL, NULL,
       NULL, NULL, NULL,
       false, NULL, NULL, NULL
FROM odometer_entries o;

COMMENT ON VIEW log_entries IS
    'The refuels, the maintenances, the expenses, the incomes, and the odometer entries of each vehicle, with the names of their reference items.';

COMMIT;
