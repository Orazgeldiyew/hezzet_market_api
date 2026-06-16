BEGIN;

ALTER TABLE stock_reservations
    DROP CONSTRAINT IF EXISTS stock_reservations_sale_id_fkey;
ALTER TABLE stock_reservations
    ADD CONSTRAINT stock_reservations_sale_id_fkey
    FOREIGN KEY (sale_id) REFERENCES sales(id);

ALTER TABLE sale_returns
    DROP CONSTRAINT IF EXISTS sale_returns_sale_id_fkey;
ALTER TABLE sale_returns
    ADD CONSTRAINT sale_returns_sale_id_fkey
    FOREIGN KEY (sale_id) REFERENCES sales(id);

ALTER TABLE supplier_debts
    DROP CONSTRAINT IF EXISTS supplier_debts_purchase_id_fkey;
ALTER TABLE supplier_debts
    ADD CONSTRAINT supplier_debts_purchase_id_fkey
    FOREIGN KEY (purchase_id) REFERENCES purchase_orders(id);

COMMIT;
