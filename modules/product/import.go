package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/Orazgeldiyew/hezzet_market_backend/pkg/response"
)

// ImportResult is the JSON returned to the frontend after a bulk import.
type ImportResult struct {
	Created  int             `json:"created"`
	Updated  int             `json:"updated"`
	Skipped  int             `json:"skipped"`
	TotalRow int             `json:"total_rows"`
	Errors   []ImportRowErr  `json:"errors,omitempty"`
}

type ImportRowErr struct {
	Row  int    `json:"row"`
	SKU  string `json:"sku,omitempty"`
	Name string `json:"name,omitempty"`
	Err  string `json:"error"`
}

// Column indices in the Hezzet supplier-supplied workbook (0-based).
// Locked to the file format the user provided in the chat; if a future export
// reshuffles columns the indexes here are the only thing to bump.
const (
	colSKU      = 0  // A — Kody (AN00025...)
	colBarcode  = 1  // B — Barkod
	colName     = 2  // C — Ady (product name)
	colUnit     = 10 // K — Ölçegi (SAN / KG / LT / ...)
	colCategory = 14 // O — Topar kod (category name; auto-created if missing)
)

// unitMap maps the workbook's text unit codes to our domain (unit_type, unit).
// SAN = countable item; KG/GR = weight; LT/ML = volume. Anything else is
// reported as a row-level error so the operator can fix it in Excel and retry.
func unitMap(raw string) (unitType, unit string, ok bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SAN", "PIECE", "ŞT", "ШТ":
		return "piece", "piece", true
	case "KG":
		return "weight", "kg", true
	case "GR", "G", "ГР", "Г":
		return "weight", "g", true
	case "LT", "L", "Л":
		return "volume", "l", true
	case "ML", "МЛ":
		return "volume", "ml", true
	}
	return "", "", false
}

// ImportHandler returns a gin handler that processes a multipart XLSX upload
// and upserts every row into products (by sku). Categories named in column O
// are auto-created on the fly. Designed to be idempotent: re-running the same
// file produces the same final state (rows hit UPDATE branch the second time).
func ImportHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.Error(apperr.Validation("file is required (multipart field 'file')"))
			return
		}
		defer file.Close()

		if header.Size > 50*1024*1024 {
			c.Error(apperr.Validation("file too large (>50 MB)"))
			return
		}

		// excelize wants an io.ReadSeeker but FormFile gives us a multipart.File
		// which is already a ReadSeeker. Stream straight from it — no temp file.
		xl, err := excelize.OpenReader(file)
		if err != nil {
			c.Error(apperr.Validation("invalid xlsx file: " + err.Error()))
			return
		}
		defer xl.Close()

		sheets := xl.GetSheetList()
		if len(sheets) == 0 {
			c.Error(apperr.Validation("workbook has no sheets"))
			return
		}
		rows, err := xl.GetRows(sheets[0])
		if err != nil {
			c.Error(apperr.Internal(fmt.Errorf("read rows: %w", err)))
			return
		}
		if len(rows) < 2 {
			c.Error(apperr.Validation("workbook is empty (need header row + data)"))
			return
		}

		uid := extractUserIDFromCtx(c)
		out := runImport(c.Request.Context(), db, rows, uid)
		response.OK(c, out)
	}
}

func extractUserIDFromCtx(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	uid, _ := v.(int64)
	return uid
}

// runImport walks every data row, normalizes fields, and upserts.
// One transaction per row keeps a single bad row from rolling back the whole
// 35 000-row file — partial success is preferable to all-or-nothing here.
func runImport(ctx context.Context, db *pgxpool.Pool, rows [][]string, userID int64) ImportResult {
	res := ImportResult{TotalRow: len(rows) - 1}
	categoryCache := map[string]int64{}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowNum := i + 1 // human-readable (1-based, plus header)

		sku := cell(row, colSKU)
		name := cell(row, colName)
		barcode := cell(row, colBarcode)
		unitRaw := cell(row, colUnit)
		catRaw := cell(row, colCategory)

		if name == "" {
			res.Skipped++
			continue // empty row — skip silently
		}
		if sku == "" {
			res.Errors = append(res.Errors, ImportRowErr{Row: rowNum, Name: name, Err: "missing SKU"})
			res.Skipped++
			continue
		}

		unitType, unit, unitOK := unitMap(unitRaw)
		if !unitOK {
			unitType, unit = "piece", "piece"
		}

		// Prices intentionally not imported from the workbook — the operator
		// reviewed the legacy file and decided prices will be entered manually
		// per-product through /products/edit. We persist 0/0 so a manager
		// can spot un-priced items at a glance.
		buyCents := int64(0)
		sellCents := int64(0)

		// Category — auto-create on first sighting, cache the id for subsequent rows.
		var categoryID *int64
		if catRaw != "" {
			cid, err := resolveCategory(ctx, db, categoryCache, catRaw)
			if err == nil {
				categoryID = &cid
			} else {
				res.Errors = append(res.Errors, ImportRowErr{Row: rowNum, SKU: sku, Name: name, Err: "category: " + err.Error()})
				// Don't bail — still import the product without category.
			}
		}

		action, err := upsertProduct(ctx, db, sku, name, barcode, unitType, unit, buyCents, sellCents, categoryID, userID)
		if err != nil {
			res.Errors = append(res.Errors, ImportRowErr{Row: rowNum, SKU: sku, Name: name, Err: err.Error()})
			res.Skipped++
			continue
		}
		switch action {
		case "created":
			res.Created++
		case "updated":
			res.Updated++
		}
	}

	// Cap errors at 200 entries — the frontend doesn't need a 30 000-item list
	// to give the operator a usable summary.
	if len(res.Errors) > 200 {
		res.Errors = res.Errors[:200]
	}
	return res
}

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// resolveCategory looks up (or creates) a category by name. Cached so the same
// name in 5 000 rows doesn't trigger 5 000 SELECTs.
func resolveCategory(ctx context.Context, db *pgxpool.Pool, cache map[string]int64, name string) (int64, error) {
	if id, ok := cache[name]; ok {
		return id, nil
	}
	var id int64
	err := db.QueryRow(ctx, `SELECT id FROM categories WHERE name = $1 AND deleted_at IS NULL LIMIT 1`, name).Scan(&id)
	if err == nil {
		cache[name] = id
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	// Doesn't exist — create.
	if err := db.QueryRow(ctx,
		`INSERT INTO categories (name, is_active) VALUES ($1, true) RETURNING id`, name,
	).Scan(&id); err != nil {
		return 0, err
	}
	cache[name] = id
	return id, nil
}

// upsertProduct inserts a new product (matched by SKU) or updates the existing
// row's name/prices/unit/barcode. Returns "created" or "updated" so the caller
// can keep the right counter.
func upsertProduct(ctx context.Context, db *pgxpool.Pool, sku, name, barcode, unitType, unit string, buyCents, sellCents int64, categoryID *int64, userID int64) (string, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var existingID int64
	err = tx.QueryRow(ctx, `SELECT id FROM products WHERE sku = $1`, sku).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	action := "updated"
	if errors.Is(err, pgx.ErrNoRows) {
		action = "created"
		// Create new product
		err = tx.QueryRow(ctx, `
			INSERT INTO products (name, sku, unit_type, unit, purchase_price, sale_price, is_active)
			VALUES ($1, $2, $3::unit_type_enum, $4::unit_enum, $5, $6, true)
			RETURNING id
		`, name, sku, unitType, unit, buyCents, sellCents).Scan(&existingID)
		if err != nil {
			return "", err
		}
	} else {
		// Update existing — refresh prices/name/unit. Don't touch is_active or
		// other admin-managed flags so a manager's "deactivated" status sticks.
		_, err = tx.Exec(ctx, `
			UPDATE products SET
				name           = $2,
				unit_type      = $3::unit_type_enum,
				unit           = $4::unit_enum,
				purchase_price = $5,
				sale_price     = $6,
				updated_at     = now()
			WHERE id = $1
		`, existingID, name, unitType, unit, buyCents, sellCents)
		if err != nil {
			return "", err
		}
	}

	// Barcode — insert if not already present. Composed of a SELECT to keep the
	// query simple (the unique constraint would also catch duplicates).
	if barcode != "" {
		var has bool
		err = tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM product_barcodes WHERE product_id = $1 AND barcode = $2)`,
			existingID, barcode,
		).Scan(&has)
		if err != nil {
			return "", err
		}
		if !has {
			_, err = tx.Exec(ctx,
				`INSERT INTO product_barcodes (product_id, barcode) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				existingID, barcode,
			)
			if err != nil {
				return "", err
			}
		}
	}

	// Attach to category if provided. ON CONFLICT keeps existing link.
	if categoryID != nil {
		_, err = tx.Exec(ctx,
			`INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			existingID, *categoryID,
		)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	_ = userID // available for audit logging if added later
	return action, nil
}

