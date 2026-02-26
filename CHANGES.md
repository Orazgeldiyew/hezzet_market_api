# Changelog

## Recent Changes

---

### Committed — `50c490c` — Product measurement enums

#### Migration `026_product_measurement_enums`
- Created PostgreSQL enum types `unit_type_enum` (`piece`, `weight`, `volume`) and `unit_enum` (`piece`, `kg`, `g`, `l`, `ml`)
- Applied to `products.unit` and `products.unit_type` columns

#### `modules/product/model.go`
- Replaced raw `string` fields with typed Go enums `UnitType` and `Unit`
- `UnitType`: `piece | weight | volume`
- `Unit`: `piece | kg | g | l | ml`
- `CreateRequest`: added `binding:"required,oneof=..."` validation on both enum fields; removed duplicate `UnitType` field
- `UpdateRequest`: same cleanup, added `binding:"omitempty,oneof=..."` on both enum pointers

#### `modules/product/repository.go`
- SQL queries now use explicit `::unit_enum` / `::unit_type_enum` casts (pgx v5 doesn't auto-convert custom enum OIDs)
- All scan calls read enums into a plain `string` then cast to typed Go type
- Added `ptrUnitToString` / `ptrUnitTypeToString` helpers for nullable UPDATE parameters

#### `modules/product/service.go`
- Added `ValidateUnit(unitType, unit)` — enforces business rules:
  - `piece` → unit must be `piece`
  - `weight` → unit must be `kg` or `g`
  - `volume` → unit must be `l` or `ml`

#### `modules/product/routes.go`
- Removed `PaginationMiddleware()` from the product list route (handled manually in handler)

#### `modules/product/handler.go`
- Fixed offset recalculation when `limit` query param overrides the default

---

### Uncommitted (working tree)

#### Migration `027_product_photo` _(untracked)_
- New table `product_photos (id, product_id, ext, created_at)` — stores a record per upload
- New column `products.photo_path TEXT` — holds the relative stored path of the current photo (e.g. `photos/42.jpg`)

#### `config/config.go`
- Added `UploadsDir` (env: `UPLOADS_DIR`, default `./uploads`) — local directory where uploaded files are stored
- Added `PublicBaseURL` (env: `PUBLIC_BASE_URL`, default `http://localhost:8080`) — used to build public-facing photo URLs

#### `server/router.go`
- Serves `GET /uploads/*` as a static directory (no auth required) using `r.Static("/uploads", uploadsDir)`
- Passes `UploadsDir` and `PublicBaseURL` to `product.RegisterRoutes` and `sale.RegisterRoutes`

#### `modules/product/model.go`
- Added `PhotoPath *string` and `PhotoURL *string` fields to `Product`

#### `modules/product/repository.go`
- `NewRepository` now takes `baseURL string`
- Added `photoURL(path *string) *string` helper — converts stored relative path to full public URL
- All SELECT queries include `photo_path`; `GetByID`, `Update`, and `List` populate `PhotoPath` and `PhotoURL`
- Added `InsertPhoto(ctx, productID, ext)` — inserts a `product_photos` record and returns the new photo ID
- Added `UpdatePhotoPath(ctx, productID, path)` — sets `products.photo_path`

#### `modules/product/service.go`
- `NewService` now takes `uploadsDir string`
- Added `SavePhoto(ctx, productID, fh, currentPhotoPath)`:
  - Validates file size (max 5 MB)
  - Sniffs MIME type (allowed: `image/jpeg`, `image/png`, `image/webp`)
  - Validates file extension (`.jpg`, `.jpeg`, `.png`, `.webp`)
  - Inserts a `product_photos` record to get a stable numeric ID
  - Stores file at `{uploadsDir}/photos/{photoID}{ext}`
  - Path-traversal guard: ensures resolved path stays inside `uploadsDir`
  - Deletes the previous file on success (best-effort)
- `Create(ctx, req, fh)` — accepts an optional `*multipart.FileHeader`; calls `SavePhoto` then `UpdatePhotoPath` if a file was provided
- Added `UploadPhoto(ctx, id, fh)` — fetches existing product, calls `SavePhoto`, updates DB, returns refreshed product

#### `modules/product/handler.go`
- `Create` now handles `multipart/form-data`:
  - Reads `data` field (JSON string) instead of a JSON body
  - Manually validates required fields (`name`, `unit_type`, `unit`)
  - Reads optional `file` field and passes it to `svc.Create`
- Added `UploadPhoto` handler — `POST /api/products/:id/photo`
  - Requires `file` form field
  - Returns updated `Product`

#### `modules/product/routes.go`
- `RegisterRoutes` now takes `uploadsDir, publicBaseURL string`
- Added `write.POST("/:id/photo", h.UploadPhoto)`

#### `modules/sale/model.go`
- `SaleItem` gains `ProductName string` and `ProductPhotoURL *string` fields (populated on read)

#### `modules/sale/repository.go`
- `NewRepository` now takes `baseURL string`; added `photoURL` helper (same pattern as product repo)
- `GetByID` JOIN query now also fetches `p.name, p.photo_path` from `products` and populates the new fields
- `CreateSale` no longer collects `SaleItem` rows inline; instead calls `GetByID` after commit to return items with product name + photo URL

#### `modules/sale/routes.go`
- `RegisterRoutes` now takes `publicBaseURL string` and passes it to `NewRepository`
