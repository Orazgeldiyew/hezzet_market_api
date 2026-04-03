package product

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	baseURL string // PUBLIC_BASE_URL for building photo URLs
}

func NewRepository(db *pgxpool.Pool, baseURL string) *Repository {
	return &Repository{db: db, baseURL: baseURL}
}

// photoURL converts a nullable DB photo_path to a full public URL.
func (r *Repository) photoURL(path *string) *string {
	if path == nil || *path == "" || r.baseURL == "" {
		return nil
	}
	u := r.baseURL + "/uploads/" + *path
	return &u
}

func (r *Repository) Create(ctx context.Context, p *Product) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := `
		INSERT INTO products (name, sku, unit, purchase_price, sale_price, is_active, unit_type)
		VALUES ($1,$2,$3::unit_enum,$4,$5,$6,$7::unit_type_enum)
		RETURNING id, unit_scale, created_at, updated_at
	`
	// Pass typed enums as plain strings — pgx sends them as text which PostgreSQL
	// accepts for enum parameters when combined with an explicit cast in the query.
	if err := tx.QueryRow(ctx, q,
		p.Name, p.SKU, string(p.Unit), p.PurchasePrice, p.SalePrice, p.IsActive, string(p.UnitType),
	).Scan(&p.ID, &p.UnitScale, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return err
	}

	for _, bc := range p.Barcodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_barcodes (product_id, barcode) VALUES ($1, $2)`,
			p.ID, bc,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Product, error) {
	q := `
		SELECT id, name, sku, unit, purchase_price, sale_price, is_active, unit_type, unit_scale, photo_path, created_at, updated_at
		FROM products WHERE id=$1 AND is_active=true
	`
	var p Product
	// Scan PostgreSQL enum columns into plain strings first, then cast to typed enums.
	// pgx v5 does not auto-convert unknown enum OIDs to custom Go string types.
	var unitStr, unitTypeStr string
	var photoPath *string
	err := r.db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.SKU, &unitStr,
		&p.PurchasePrice, &p.SalePrice, &p.IsActive, &unitTypeStr, &p.UnitScale, &photoPath, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.Unit = Unit(unitStr)
	p.UnitType = UnitType(unitTypeStr)
	p.PhotoPath = photoPath
	p.PhotoURL = r.photoURL(photoPath)
	p.Barcodes, err = r.loadBarcodes(ctx, id)
	return p, err
}

func (r *Repository) GetByBarcode(ctx context.Context, barcode string) (Product, error) {
	var productID int64
	err := r.db.QueryRow(ctx,
		`SELECT product_id FROM product_barcodes WHERE barcode = $1`, barcode,
	).Scan(&productID)
	if err != nil {
		return Product{}, err
	}
	return r.GetByID(ctx, productID)
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (Product, error) {
	q := `
		UPDATE products SET
			name          = COALESCE($2, name),
			sku           = COALESCE($3, sku),
			unit          = COALESCE($4::unit_enum, unit),
			purchase_price = COALESCE($5, purchase_price),
			sale_price    = COALESCE($6, sale_price),
			is_active     = COALESCE($7, is_active),
			unit_type     = COALESCE($8::unit_type_enum, unit_type),
			updated_at    = now()
		WHERE id=$1
		RETURNING id, name, sku, unit, purchase_price, sale_price, is_active, unit_type, unit_scale, photo_path, created_at, updated_at
	`
	// Convert *UnitType and *Unit to *string so pgx sends NULL when nil,
	// which COALESCE correctly interprets as "keep existing value".
	unitParam := ptrUnitToString(req.Unit)
	unitTypeParam := ptrUnitTypeToString(req.UnitType)

	var p Product
	var unitStr, unitTypeStr string
	var photoPath *string
	err := r.db.QueryRow(ctx, q,
		id, req.Name, req.SKU, unitParam,
		req.PurchasePrice, req.SalePrice, req.IsActive, unitTypeParam,
	).Scan(
		&p.ID, &p.Name, &p.SKU, &unitStr,
		&p.PurchasePrice, &p.SalePrice, &p.IsActive, &unitTypeStr, &p.UnitScale, &photoPath, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.Unit = Unit(unitStr)
	p.UnitType = UnitType(unitTypeStr)
	p.PhotoPath = photoPath
	p.PhotoURL = r.photoURL(photoPath)

	if req.Barcodes != nil {
		if err := r.ReplaceBarcodes(ctx, id, *req.Barcodes); err != nil {
			return p, err
		}
	}

	p.Barcodes, err = r.loadBarcodes(ctx, id)
	return p, err
}

// ptrUnitTypeToString converts *UnitType → *string for pgx parameter passing.
func ptrUnitTypeToString(v *UnitType) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

// ptrUnitToString converts *Unit → *string for pgx parameter passing.
func ptrUnitToString(v *Unit) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func (r *Repository) List(ctx context.Context, limit, offset int, orderBy, orderDir, qstr string) ([]Product, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// total count
	countSQL := `
		SELECT COUNT(DISTINCT p.id) FROM products p
		LEFT JOIN product_barcodes pb ON pb.product_id = p.id
		WHERE p.is_active = true
		  AND ($1 = '' OR
		       p.name ILIKE '%' || $1 || '%' OR
		       p.sku ILIKE '%' || $1 || '%' OR
		       pb.barcode ILIKE '%' || $1 || '%')
	`
	var total int
	if err := r.db.QueryRow(ctx, countSQL, qstr).Scan(&total); err != nil {
		return nil, 0, err
	}

	// whitelist order column
	col := "p.created_at"
	switch orderBy {
	case "name":
		col = "p.name"
	case "created_at":
		col = "p.created_at"
	}

	dir := "DESC"
	if orderDir == "asc" {
		dir = "ASC"
	}

	sql := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.name, p.sku, p.unit, p.purchase_price, p.sale_price, p.is_active, p.unit_type, p.unit_scale, p.photo_path, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN product_barcodes pb ON pb.product_id = p.id
		WHERE p.is_active = true
		  AND ($1 = '' OR
		       p.name ILIKE '%%' || $1 || '%%' OR
		       p.sku ILIKE '%%' || $1 || '%%' OR
		       pb.barcode ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, col, dir)

	rows, err := r.db.Query(ctx, sql, qstr, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ids []int64
	var out []Product
	for rows.Next() {
		var p Product
		var unitStr, unitTypeStr string
		var photoPath *string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.SKU, &unitStr,
			&p.PurchasePrice, &p.SalePrice, &p.IsActive, &unitTypeStr, &p.UnitScale, &photoPath, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		p.Unit = Unit(unitStr)
		p.UnitType = UnitType(unitTypeStr)
		p.PhotoPath = photoPath
		p.PhotoURL = r.photoURL(photoPath)
		ids = append(ids, p.ID)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// batch-load barcodes
	if len(ids) > 0 {
		bcMap, err := r.loadBarcodesMap(ctx, ids)
		if err != nil {
			return nil, 0, err
		}
		for i := range out {
			out[i].Barcodes = bcMap[out[i].ID]
		}
	}

	return out, total, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE products SET is_active=false, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// InsertPhoto creates a record in product_photos and returns the new photo ID.
func (r *Repository) InsertPhoto(ctx context.Context, productID int64, ext string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx,
		`INSERT INTO product_photos (product_id, ext) VALUES ($1, $2) RETURNING id`,
		productID, ext,
	).Scan(&id)
	return id, err
}

// UpdatePhotoPath sets the current photo_path on a product.
func (r *Repository) UpdatePhotoPath(ctx context.Context, productID int64, path *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE products SET photo_path = $1, updated_at = now() WHERE id = $2 AND is_active = true`,
		path, productID,
	)
	return err
}

func IsNoRows(err error) bool { return err == pgx.ErrNoRows }

//
// ===== Product ↔ Barcodes (one-to-many) =====
//

func (r *Repository) loadBarcodes(ctx context.Context, productID int64) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT barcode FROM product_barcodes WHERE product_id = $1 ORDER BY id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			return nil, err
		}
		out = append(out, bc)
	}
	if out == nil {
		out = []string{}
	}
	return out, rows.Err()
}

func (r *Repository) loadBarcodesMap(ctx context.Context, productIDs []int64) (map[int64][]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT product_id, barcode FROM product_barcodes WHERE product_id = ANY($1) ORDER BY id`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64][]string, len(productIDs))
	for rows.Next() {
		var pid int64
		var bc string
		if err := rows.Scan(&pid, &bc); err != nil {
			return nil, err
		}
		m[pid] = append(m[pid], bc)
	}
	// ensure every requested ID has an entry
	for _, id := range productIDs {
		if _, ok := m[id]; !ok {
			m[id] = []string{}
		}
	}
	return m, rows.Err()
}

func (r *Repository) ReplaceBarcodes(ctx context.Context, productID int64, barcodes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM product_barcodes WHERE product_id = $1`, productID); err != nil {
		return err
	}
	for _, bc := range barcodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_barcodes (product_id, barcode) VALUES ($1, $2)`,
			productID, bc,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

//
// ===== Product ↔ Categories (many-to-many) =====
//

func (r *Repository) ListCategories(ctx context.Context, productID int64) ([]CategoryBrief, error) {
	q := `
		SELECT c.id, c.name
		FROM product_categories pc
		JOIN categories c ON c.id = pc.category_id
		WHERE pc.product_id = $1 AND c.is_active = true
		ORDER BY c.name
	`
	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CategoryBrief
	for rows.Next() {
		var b CategoryBrief
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) ReplaceCategories(ctx context.Context, productID int64, categoryIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM product_categories WHERE product_id=$1`, productID); err != nil {
		return err
	}

	for _, cid := range categoryIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_categories (product_id, category_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			productID, cid,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) RemoveCategory(ctx context.Context, productID, categoryID int64) (bool, error) {
	ct, err := r.db.Exec(ctx,
		`DELETE FROM product_categories WHERE product_id=$1 AND category_id=$2`,
		productID, categoryID,
	)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *Repository) CountActiveCategoriesByIDs(ctx context.Context, ids []int64) (int, error) {
	q := `SELECT COUNT(*) FROM categories WHERE is_active=true AND id = ANY($1)`
	var n int
	if err := r.db.QueryRow(ctx, q, ids).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
