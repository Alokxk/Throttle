DROP INDEX IF EXISTS idx_clients_key_prefix;

ALTER TABLE clients
DROP COLUMN IF EXISTS api_key_hash,
DROP COLUMN IF EXISTS key_prefix;
