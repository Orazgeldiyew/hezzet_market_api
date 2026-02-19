BEGIN;

DROP TABLE IF EXISTS warehouse_item_details;
DROP TABLE IF EXISTS warehouse_items;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'movement_type') THEN
    DROP TYPE movement_type;
  END IF;
END
$$;

COMMIT;
