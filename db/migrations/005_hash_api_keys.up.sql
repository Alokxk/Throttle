ALTER TABLE clients
ADD COLUMN IF NOT EXISTS key_prefix VARCHAR(10),
ADD COLUMN IF NOT EXISTS api_key_hash VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_clients_key_prefix
ON clients(key_prefix);