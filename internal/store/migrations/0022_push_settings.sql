-- Milestone 20: per-project push settings. A serialized
-- moth.projectconfig.v1.StoredPush protobuf document on the project row,
-- exactly like paywall_pb — but plain config with no revision history: it
-- carries only the enabled switch and the Web Push VAPID PUBLIC key (the
-- private key stays with the developer's sender and never touches moth).
-- Empty means "push never configured" (disabled).
ALTER TABLE projects ADD COLUMN push_pb BLOB NOT NULL DEFAULT X'';
