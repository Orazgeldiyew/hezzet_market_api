CREATE TABLE receipt_settings (
    id           INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    shop_name    TEXT,
    shop_address TEXT,
    shop_phone   TEXT,
    logo_path    TEXT,
    footer       TEXT,
    template     TEXT,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO receipt_settings (id) VALUES (1);
