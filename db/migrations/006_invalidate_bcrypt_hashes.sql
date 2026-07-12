-- Switching API key hashing from bcrypt to SHA-256. Existing bcrypt
-- hashes can't be converted (one-way), so clear them; affected rows
-- fall back to the existing plaintext api_key comparison path until
-- the client re-registers.
UPDATE clients SET api_key_hash = NULL WHERE api_key_hash IS NOT NULL;
