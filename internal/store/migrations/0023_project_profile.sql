-- Milestone 22: per-project setup profile. A serialized
-- moth.projectconfig.v1.StoredProfile protobuf document on the project row,
-- exactly like push_pb — plain config with no revision history: the creation
-- wizard's answers (platforms, sign-in / monetization / push intent, the
-- checklist-dismissed flag). Intent only, never a second source of config
-- truth. Empty means "no profile" (a project created before the wizard);
-- every adaptive surface then behaves exactly as pre-milestone.
ALTER TABLE projects ADD COLUMN profile_pb BLOB NOT NULL DEFAULT X'';
