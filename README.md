# Hezzet Market Backend

A clean, modular REST API backend for a small market POS/admin system built with Go, Gin, PostgreSQL (pgx/pgxpool), and Swagger.

## Features

- **Category Management**: Hierarchical categories with parent-child relationships
- **Product Management**: Complete CRUD with category associations and stock tracking
- **Supplier Management**: Supplier information with soft delete support
- **Product-Category Linking**: Many-to-many relationship management
- **Swagger Documentation**: Auto-generated API docs at `/swagger/index.html`
- **Clean Architecture**: Modular design with repository, service, and handler layers
- **Consistent Error Handling**: Unified error responses with custom error types
- **Pagination**: All list endpoints support limit/offset pagination
- **Search**: Full-text search across relevant fields

## Tech Stack

- **Go 1.25.6**
- **Gin** - HTTP web framework
- **PostgreSQL** with **pgx/pgxpool** - Database and connection pooling
- **Swaggo** - Swagger documentation generation
- **godotenv** - Environment variable management

## Project Structure

```
hezzet_market_backend/
├── config/              # Configuration loading
├── migrations/          # Database schema migrations
├── modules/             # Business logic modules
│   ├── category/       # Category module
│   ├── product/        # Product module
│   └── supplier/       # Supplier module
├── pkg/                # Shared packages
│   ├── database/       # Database connection pool
│   ├── errors/         # Custom error types
│   └── response/       # HTTP response helpers
├── server/             # HTTP server setup
│   ├── router.go       # Route registration
│   └── middleware.go   # Middleware (error handling)
├── docs/               # Generated Swagger docs
├── main.go             # Application entry point
└── API_TESTING.md      # API testing guide with curl examples
```

## Module Structure

Each module follows a consistent pattern:

```
modules/<module>/
├── model.go       # Data models and DTOs
├── repository.go  # Database access layer
├── service.go     # Business logic layer
├── handler.go     # HTTP handlers
└── routes.go      # Route registration
```

## Database Schema

### Tables

- **categories**: Hierarchical product categories with `parent_id` FK
- **suppliers**: Supplier information
- **products**: Product master data
- **product_categories**: Many-to-many junction table

See [migrations/001_schema.sql](migrations/001_schema.sql) for complete schema.

## Setup

### Prerequisites

- Go 1.25.6+
- PostgreSQL 12+
- swag CLI tool

### Install swag

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Environment Variables

Create a `.env` file:

```env
ENV=dev
HTTP_ADDR=:8080
DB_DSN=postgres://user:password@localhost:5432/hezzet_market?sslmode=disable
```

### Database Setup

1. Create PostgreSQL database:
```bash
createdb hezzet_market
```

2. Run migrations:
```bash
psql -U postgres -d hezzet_market -f migrations/001_schema.sql
```

### Generate Swagger Docs

```bash
swag init -g main.go -o docs --parseInternal --parseDependency
```

### Run the Application

```bash
# Install dependencies
go mod download

# Run the server
go run main.go
```

The server will start on `http://localhost:8080`

## API Documentation

### Swagger UI
Access interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

### Testing with curl
See [API_TESTING.md](API_TESTING.md) for comprehensive curl examples.

## API Endpoints

### Categories
- `POST /categories` - Create category
- `GET /categories` - List categories (paginated)
- `GET /categories/tree` - Get category hierarchy
- `GET /categories/:id` - Get single category
- `PATCH /categories/:id` - Update category
- `DELETE /categories/:id` - Soft delete category

### Suppliers
- `POST /suppliers` - Create supplier
- `GET /suppliers` - List suppliers (paginated, searchable)
- `GET /suppliers/:id` - Get single supplier
- `PATCH /suppliers/:id` - Update supplier
- `DELETE /suppliers/:id` - Soft delete supplier

### Products
- `POST /products` - Create product
- `GET /products` - List products (paginated, searchable)
- `GET /products/:id` - Get single product
- `PATCH /products/:id` - Update product
- `GET /products/:id/card` - Get product card (with stock & categories)
- `GET /products/:id/categories` - Get product categories
- `PUT /products/:id/categories` - Set/replace product categories
- `DELETE /products/:id/categories/:categoryId` - Remove category from product

### Other
- `GET /health` - Health check
- `GET /swagger/*any` - Swagger documentation

## Response Format

### Success Response
```json
{
  "success": true,
  "data": { ... }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

## Design Principles

1. **No Import Cycles**: Modules don't import from `server` package
2. **Repository Pattern**: Database logic isolated in repositories
3. **Service Layer**: Business logic in services
4. **Thin Handlers**: Handlers only parse input and call services
5. **Consistent Errors**: All errors use `pkg/errors.AppError`
6. **Consistent Responses**: All responses use `pkg/response` helpers
7. **Soft Deletes**: Records marked inactive instead of deleted
8. **Pagination**: Default limit 50, max 200

## Building for Production

```bash
# Build binary
go build -o hezzet_market_backend

# Run binary
./hezzet_market_backend
```

## Development

### Code Generation

After changing Swagger annotations, regenerate docs:
```bash
swag init -g main.go -o docs --parseInternal --parseDependency
```

### Code Formatting

```bash
go fmt ./...
```

### Verify Build

```bash
go build
```

## License

Proprietary - All rights reserved
