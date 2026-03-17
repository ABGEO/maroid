BEGIN;

CREATE TYPE source_type AS ENUM ('plant', 'environment');

ALTER TABLE measurements
    ADD COLUMN source_type source_type NOT NULL DEFAULT 'plant';

ALTER TABLE measurements
    RENAME COLUMN plant_id TO source_id;

DROP INDEX idx_plant_metric_time;
CREATE INDEX idx_source_metric_time ON measurements (source_type, source_id, metric_type, time DESC);

COMMIT;
