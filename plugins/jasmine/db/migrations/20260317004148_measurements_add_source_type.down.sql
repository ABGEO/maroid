BEGIN;

DROP INDEX idx_source_metric_time;
CREATE INDEX idx_plant_metric_time ON measurements (source_id, metric_type, time DESC);

ALTER TABLE measurements
    RENAME COLUMN source_id TO plant_id;

ALTER TABLE measurements
    DROP COLUMN source_type;

DROP TYPE source_type;

COMMIT;
