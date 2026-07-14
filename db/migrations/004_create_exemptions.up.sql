CREATE TABLE IF NOT EXISTS exemptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID REFERENCES clients(id) NOT NULL,
    identifier  VARCHAR(255) NOT NULL,
    reason      VARCHAR(500),
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(client_id, identifier)
);

CREATE INDEX IF NOT EXISTS idx_exemptions_lookup
ON exemptions(client_id, identifier);