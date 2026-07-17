-- Per-project config storage moves from JSON TEXT to protobuf BLOBs
-- (moth.storage.v1: StoredTheme / StoredPaywall / StoredCopy). The new *_pb
-- columns hold the proto-encoded document; X'' (empty) keeps the existing
-- convention: built-in defaults, nothing customized.
--
-- The legacy TEXT columns (projects.theme/paywall/copy and
-- theme_revisions.theme / paywall_revisions.paywall / copy_revisions.copy)
-- are FROZEN from this migration on: a one-time Go backfill (run right after
-- migrations apply, see store.backfillProtoConfigs) parses every non-empty
-- legacy JSON document, re-encodes it as protobuf into the *_pb column and
-- clears the TEXT column to ''; all live code paths read and write only the
-- *_pb columns, inserting '' into the NOT NULL legacy columns.

ALTER TABLE projects ADD COLUMN theme_pb   BLOB NOT NULL DEFAULT X'';
ALTER TABLE projects ADD COLUMN paywall_pb BLOB NOT NULL DEFAULT X'';
ALTER TABLE projects ADD COLUMN copy_pb    BLOB NOT NULL DEFAULT X'';

ALTER TABLE theme_revisions   ADD COLUMN theme_pb   BLOB NOT NULL DEFAULT X'';
ALTER TABLE paywall_revisions ADD COLUMN paywall_pb BLOB NOT NULL DEFAULT X'';
ALTER TABLE copy_revisions    ADD COLUMN copy_pb    BLOB NOT NULL DEFAULT X'';
