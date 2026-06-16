BEGIN;

-- Three tables had FOREIGN KEY references with no ON DELETE clause, so a
-- DELETE on the parent (sales, purchase_orders) would either fail with a FK
-- violation or — if the app inserts rows directly — leave orphans. Adding
-- explicit cascade behavior aligns each FK with how the parent is actually
-- managed.

-- stock_reservations: a reservation only exists for the sale it belongs to.
-- If a sale is hard-deleted, its reservations must go with it.
ALTER TABLE stock_reservations
    DROP CONSTRAINT IF EXISTS stock_reservations_sale_id_fkey;
ALTER TABLE stock_reservations
    ADD CONSTRAINT stock_reservations_sale_id_fkey
    FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE;

-- sale_returns: same logic — returns are children of a sale.
ALTER TABLE sale_returns
    DROP CONSTRAINT IF EXISTS sale_returns_sale_id_fkey;
ALTER TABLE sale_returns
    ADD CONSTRAINT sale_returns_sale_id_fkey
    FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE;

-- supplier_debts: debts can outlive their originating purchase. If a PO is
-- hard-deleted, keep the debt row but drop the dangling pointer.
ALTER TABLE supplier_debts
    DROP CONSTRAINT IF EXISTS supplier_debts_purchase_id_fkey;
ALTER TABLE supplier_debts
    ADD CONSTRAINT supplier_debts_purchase_id_fkey
    FOREIGN KEY (purchase_id) REFERENCES purchase_orders(id) ON DELETE SET NULL;

COMMIT;
