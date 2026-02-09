# Client Module Implementation

## Overview
Complete implementation of the `client` module for customer management in the Hezzet Market POS system.

## Files Created

### 1. modules/client/model.go
- `Client` struct with all required fields (int64 ID, name, phone, email, is_active, timestamps)
- `CreateRequest` with validation (name required, phone/email optional)
- `UpdateRequest` with pointer fields for partial updates
- `ListResponse` for paginated responses

### 2. modules/client/repository.go
- `Create()` - Insert new client with RETURNING clause
- `GetByID()` - Fetch client by ID
- `List()` - Paginated list with search (name OR phone) and ORDER BY id DESC
- `Update()` - Partial update with COALESCE and updated_at timestamp
- `SoftDelete()` - Set is_active=false instead of deleting
- `IsNotFound()` - Helper using pgx.ErrNoRows

### 3. modules/client/service.go
- `Create()` - Business logic with unique phone constraint handling (409 Conflict)
- `Get()` - Returns 404 for not found OR inactive clients
- `List()` - Returns only active clients with pagination
- `Update()` - Handles unique phone constraint on updates
- `Delete()` - Soft delete logic
- Proper error mapping using pkg/errors helpers

### 4. modules/client/handler.go
- `Create()` - POST /clients with Swagger annotations
- `List()` - GET /clients with pagination (limit/offset/q)
- `Get()` - GET /clients/:id
- `Update()` - PATCH /clients/:id
- `Delete()` - DELETE /clients/:id (soft delete)
- All handlers use `response.OK()`, `response.Created()`, and `c.Error()`
- Complete Swagger documentation with @Tags Clients

### 5. modules/client/routes.go
- `RegisterRoutes()` function
- Route group `/clients`
- Dependency injection (repository → service → handler)

### 6. server/router.go (Updated)
- Added client module import
- Registered client routes with `client.RegisterRoutes(r, deps.DB)`

### 7. migrations/002_clients.sql
- Complete table schema with indexes
- UNIQUE constraint on phone
- Indexes on name, phone, is_active, created_at

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /clients | Create new client |
| GET | /clients | List active clients (paginated, searchable) |
| GET | /clients/:id | Get single active client |
| PATCH | /clients/:id | Update client (partial) |
| DELETE | /clients/:id | Soft delete client |

## Key Features

✅ **Soft Delete**: Sets is_active=false instead of deleting records
✅ **Unique Phone**: Returns 409 Conflict if phone already exists
✅ **Inactive Filtering**: GET endpoints return only active clients
✅ **Search**: List endpoint searches by name OR phone
✅ **Pagination**: Default limit=50, max=200, ORDER BY id DESC
✅ **Proper Error Handling**: Uses pkg/errors.NotFound, Validation, Internal
✅ **Swagger Documentation**: Complete annotations on all handlers
✅ **Production Quality**: Context propagation, int64 IDs, proper validation

## Error Codes

- `NOT_FOUND` (404) - Client not found or inactive
- `PHONE_ALREADY_EXISTS` (409) - Phone number already in use
- `VALIDATION_ERROR` (400) - Invalid input data
- `INTERNAL_ERROR` (500) - Server error

## Testing

Build verification:
```bash
go build
# ✅ Build successful, no errors
```

Code formatting:
```bash
gofmt -l modules/client/
# ✅ All files properly formatted
```

## Example Usage

### Create Client
```bash
curl -X POST http://localhost:8080/clients \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "phone": "+1234567890",
    "email": "john@example.com"
  }'
```

### List Clients
```bash
# All active clients
curl http://localhost:8080/clients

# With pagination and search
curl "http://localhost:8080/clients?limit=20&offset=0&q=john"
```

### Get Client
```bash
curl http://localhost:8080/clients/1
```

### Update Client
```bash
curl -X PATCH http://localhost:8080/clients/1 \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+9876543210",
    "email": "newemail@example.com"
  }'
```

### Delete Client (Soft)
```bash
curl -X DELETE http://localhost:8080/clients/1
```

## Integration

The client module is fully integrated into the application:
1. Routes registered in server/router.go
2. Follows exact same pattern as product/category/supplier
3. No import cycles
4. Ready for Swagger doc generation

Run `swag init -g main.go -o docs --parseInternal --parseDependency` to regenerate Swagger docs.
