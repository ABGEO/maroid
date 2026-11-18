BEGIN;

ALTER TABLE plants
    DROP COLUMN profile_photo_id;

DROP TABLE photos;
DROP TABLE moments;

COMMIT;
