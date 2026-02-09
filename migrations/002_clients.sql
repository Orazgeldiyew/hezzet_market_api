-- Clients table for customer management
CREATE TABLE IF NOT EXISTS clients (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT UNIQUE,
    email TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name);
CREATE INDEX IF NOT EXISTS idx_clients_phone ON clients(phone);
CREATE INDEX IF NOT EXISTS idx_clients_is_active ON clients(is_active);
CREATE INDEX IF NOT EXISTS idx_clients_created_at ON clients(created_at DESC);
