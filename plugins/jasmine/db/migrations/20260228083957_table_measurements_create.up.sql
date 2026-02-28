CREATE TABLE IF NOT EXISTS measurements
(
    time        TIMESTAMPTZ      NOT NULL,
    plant_id    TEXT             NOT NULL,
    metric_type TEXT             NOT NULL,
    value       DOUBLE PRECISION NOT NULL
) WITH (tsdb.hypertable);

CREATE INDEX idx_plant_metric_time ON measurements (plant_id, metric_type, time DESC);
