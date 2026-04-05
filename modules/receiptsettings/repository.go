package receiptsettings

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db       *pgxpool.Pool
	defaults Defaults
}

func NewRepository(db *pgxpool.Pool, defaults Defaults) *Repository {
	return &Repository{db: db, defaults: defaults}
}

// Get reads the singleton row and merges NULLs with env defaults.
func (r *Repository) Get(ctx context.Context) (ReceiptSettings, error) {
	var shopName, shopAddr, shopPhone, logoPath, logoWidth, logoHeight, footer, tmpl, deleteCode *string

	err := r.db.QueryRow(ctx, `
		SELECT shop_name, shop_address, shop_phone, logo_path, logo_width, logo_height, footer, template, delete_code
		FROM receipt_settings
		WHERE id = 1
	`).Scan(&shopName, &shopAddr, &shopPhone, &logoPath, &logoWidth, &logoHeight, &footer, &tmpl, &deleteCode)
	if err != nil {
		return ReceiptSettings{}, err
	}

	return ReceiptSettings{
		ShopName:    coalesce(shopName, r.defaults.ShopName),
		ShopAddress: coalesce(shopAddr, r.defaults.ShopAddress),
		ShopPhone:   coalesce(shopPhone, r.defaults.ShopPhone),
		LogoPath:    deref(logoPath),
		LogoWidth:   coalesce(logoWidth, "50mm"),
		LogoHeight:  coalesce(logoHeight, "20mm"),
		Footer:      coalesce(footer, r.defaults.Footer),
		Template:    deref(tmpl),
		DeleteCode:  coalesce(deleteCode, "0000"),
	}, nil
}

// Update sets only the provided (non-nil) fields. Empty string → NULL (revert to default).
func (r *Repository) Update(ctx context.Context, req UpdateRequest) error {
	sets := []string{}
	args := []any{}
	idx := 1

	add := func(col string, val *string) {
		if val == nil {
			return
		}
		if *val == "" {
			sets = append(sets, fmt.Sprintf("%s = NULL", col))
		} else {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
			args = append(args, *val)
			idx++
		}
	}

	add("shop_name", req.ShopName)
	add("shop_address", req.ShopAddress)
	add("shop_phone", req.ShopPhone)
	add("logo_path", req.LogoPath)
	add("logo_width", req.LogoWidth)
	add("logo_height", req.LogoHeight)
	add("footer", req.Footer)
	add("template", req.Template)
	add("delete_code", req.DeleteCode)

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = now()")

	query := fmt.Sprintf("UPDATE receipt_settings SET %s WHERE id = 1", strings.Join(sets, ", "))
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

// GetDeleteCode returns the current delete confirmation code.
func (r *Repository) GetDeleteCode(ctx context.Context) (string, error) {
	var code *string
	err := r.db.QueryRow(ctx, `SELECT delete_code FROM receipt_settings WHERE id = 1`).Scan(&code)
	if err != nil {
		return "", err
	}
	return coalesce(code, "0000"), nil
}

func coalesce(val *string, def string) string {
	if val != nil {
		return *val
	}
	return def
}

func deref(val *string) string {
	if val != nil {
		return *val
	}
	return ""
}
