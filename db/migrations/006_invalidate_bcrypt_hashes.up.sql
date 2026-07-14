-- bcrypt hashes can't be converted to SHA-256 (one-way); clear them so
-- affected rows fall back to the plaintext comparison path until re-registration.
UPDATE clients SET api_key_hash = NULL WHERE api_key_hash IS NOT NULL;
