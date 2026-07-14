CREATE TABLE IF NOT EXISTS rules (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     UUID REFERENCES clients(id) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    algorithm     VARCHAR(50) NOT NULL,
    limit_val     INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(client_id, name)
);