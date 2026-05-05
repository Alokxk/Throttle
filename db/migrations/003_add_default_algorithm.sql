ALTER TABLE clients
ADD COLUMN IF NOT EXISTS default_algorithm VARCHAR(50) NOT NULL DEFAULT 'fixed_window';