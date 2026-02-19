BEGIN;

-- warehouses must exist
DO $$
BEGIN
  IF to_regclass('public.warehouses') IS NULL THEN
    RAISE EXCEPTION 'warehouses table does not exist. Apply migration 008 first.';
  END IF;
END
$$;

-- rename legacy stock_movements away (if exists and has no warehouse_id)
DO $$
BEGIN
  IF to_regclass('public.stock_movements') IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = 'public'
        AND table_name   = 'stock_movements'
        AND column_name  = 'warehouse_id'
    ) THEN
      ALTER TABLE public.stock_movements RENAME TO stock_movements_legacy;
    END IF;
  END IF;
END
$$;

-- movement enum
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'movement_type') THEN
    CREATE TYPE movement_type AS ENUM (
      'in',
      'out',
      'transfer_in',
      'transfer_out',
      'damaged',
      'adjustment'
    );
  END IF;
END
$$;

-- balances
CREATE TABLE IF NOT EXISTS warehouse_items (
  warehouse_id BIGINT NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
  product_id   BIGINT NOT NULL REFERENCES products(id)   ON DELETE RESTRICT,
  qty_milli    BIGINT NOT NULL DEFAULT 0,
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (warehouse_id, product_id),
  CONSTRAINT warehouse_items_qty_milli_check CHECK (qty_milli >= 0)
);

-- ledger
CREATE TABLE IF NOT EXISTS warehouse_item_details (
  id              BIGSERIAL PRIMARY KEY,
  idempotency_key UUID NOT NULL,
  warehouse_id    BIGINT NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
  product_id      BIGINT NOT NULL REFERENCES products(id)   ON DELETE RESTRICT,
  delta_milli     BIGINT NOT NULL,
  type            movement_type NOT NULL,
  price_cents     BIGINT,
  worker_id       BIGINT,
  created_by      BIGINT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_warehouse_item_details_idempotency
  ON warehouse_item_details (idempotency_key, warehouse_id);

CREATE INDEX IF NOT EXISTS idx_warehouse_item_details_wh_prod
  ON warehouse_item_details (warehouse_id, product_id);

CREATE INDEX IF NOT EXISTS idx_warehouse_item_details_type
  ON warehouse_item_details (type);

CREATE INDEX IF NOT EXISTS idx_warehouse_item_details_created_at
  ON warehouse_item_details (created_at);

COMMIT;
