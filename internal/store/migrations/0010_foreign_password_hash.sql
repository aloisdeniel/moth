-- Marks a user's password_hash as produced by a foreign algorithm during a
-- migration import (bcrypt / scrypt / argon2 / pbkdf2). Empty means the
-- native moth format (argon2id): the default for every existing and
-- natively-created user. The auth path verifies a foreign hash with its
-- original algorithm, then transparently rehashes to argon2id and clears
-- this marker on the first successful sign-in.
ALTER TABLE users ADD COLUMN password_algo TEXT NOT NULL DEFAULT '';
