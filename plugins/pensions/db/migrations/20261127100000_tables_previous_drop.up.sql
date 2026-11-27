BEGIN;

-- The rows hold a hash of every member as the identity, so a corrected amount added a
-- row. The next collection or a backfill reads them again.
DROP TABLE contributions;
DROP TABLE organizations;

COMMIT;
