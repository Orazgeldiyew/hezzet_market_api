# API Testing Guide

This document provides curl command examples for testing all API endpoints.

**Base URL**: `http://localhost:8080` (adjust port as needed)

---

## Health Check

```bash
curl http://localhost:8080/health
```

---

## Category Module

### Create Category
```bash
# Create root category
curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Electronics"
  }'

# Create child category
curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptops",
    "parent_id": 1
  }'
```

### List Categories
```bash
# List all active categories
curl http://localhost:8080/categories

# List with pagination
curl "http://localhost:8080/categories?limit=10&offset=0"

# Search categories
curl "http://localhost:8080/categories?q=elect"
```

### Get Category Tree
```bash
curl http://localhost:8080/categories/tree
```

### Get Single Category
```bash
curl http://localhost:8080/categories/1
```

### Update Category
```bash
curl -X PATCH http://localhost:8080/categories/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Consumer Electronics"
  }'

# Move to different parent
curl -X PATCH http://localhost:8080/categories/2 \
  -H "Content-Type: application/json" \
  -d '{
    "parent_id": 3
  }'

# Deactivate category
curl -X PATCH http://localhost:8080/categories/1 \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'
```

### Delete Category (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/categories/1
```

---

## Supplier Module

### Create Supplier
```bash
curl -X POST http://localhost:8080/suppliers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tech Distributors Inc",
    "phone": "+1-555-0123",
    "email": "contact@techdist.com",
    "address": "123 Main St, New York, NY 10001"
  }'
```

### List Suppliers
```bash
# List all active suppliers
curl http://localhost:8080/suppliers

# List with pagination
curl "http://localhost:8080/suppliers?limit=20&offset=0"

# Search by name, phone, or email
curl "http://localhost:8080/suppliers?q=tech"
```

### Get Single Supplier
```bash
curl http://localhost:8080/suppliers/1
```

### Update Supplier
```bash
curl -X PATCH http://localhost:8080/suppliers/1 \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+1-555-9999",
    "email": "newemail@techdist.com"
  }'

# Deactivate supplier
curl -X PATCH http://localhost:8080/suppliers/1 \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'
```

### Delete Supplier (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/suppliers/1
```

---

## Product Module

### Create Product
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MacBook Pro 16",
    "sku": "MBP16-2024",
    "barcode": "123456789012",
    "unit": "pcs",
    "purchase_price": 2000.00,
    "sale_price": 2499.99
  }'
```

### List Products
```bash
# List all products
curl http://localhost:8080/products

# List with pagination
curl "http://localhost:8080/products?limit=50&offset=0"

# Search by name, SKU, or barcode
curl "http://localhost:8080/products?q=macbook"
```

### Get Single Product
```bash
curl http://localhost:8080/products/1
```

### Update Product
```bash
curl -X PATCH http://localhost:8080/products/1 \
  -H "Content-Type: application/json" \
  -d '{
    "sale_price": 2299.99,
    "purchase_price": 1900.00
  }'

# Deactivate product
curl -X PATCH http://localhost:8080/products/1 \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'
```

### Get Product Card
Get product with stock and categories:
```bash
curl http://localhost:8080/products/1/card
```

### Product Categories Management

#### Get Product Categories
```bash
curl http://localhost:8080/products/1/categories
```

#### Set/Replace Product Categories
Replace all categories for a product:
```bash
curl -X PUT http://localhost:8080/products/1/categories \
  -H "Content-Type: application/json" \
  -d '{
    "category_ids": [1, 2, 5]
  }'

# Clear all categories
curl -X PUT http://localhost:8080/products/1/categories \
  -H "Content-Type: application/json" \
  -d '{
    "category_ids": []
  }'
```

#### Remove Single Category from Product
```bash
curl -X DELETE http://localhost:8080/products/1/categories/2
```

---

## Complete Workflow Example

Here's a complete workflow to set up test data:

```bash
# 1. Create categories
curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -d '{"name": "Electronics"}'

curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -d '{"name": "Laptops", "parent_id": 1}'

curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -d '{"name": "Apple", "parent_id": 2}'

# 2. Create supplier
curl -X POST http://localhost:8080/suppliers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Apple Distributor",
    "phone": "+1-800-APPLE",
    "email": "wholesale@apple.com",
    "address": "1 Apple Park Way, Cupertino, CA"
  }'

# 3. Create product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MacBook Pro 16 M3",
    "sku": "MBP16-M3-2024",
    "barcode": "194253912345",
    "unit": "pcs",
    "purchase_price": 2200.00,
    "sale_price": 2699.99
  }'

# 4. Assign categories to product
curl -X PUT http://localhost:8080/products/1/categories \
  -H "Content-Type: application/json" \
  -d '{"category_ids": [1, 2, 3]}'

# 5. Get product card
curl http://localhost:8080/products/1/card

# 6. List category tree
curl http://localhost:8080/categories/tree
```

---

## Swagger UI

Access interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

---

## Response Format

All responses follow this structure:

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
    "message": "Human readable error message"
  }
}
```

---

## Common Error Codes

- `NOT_FOUND` - Resource not found (404)
- `VALIDATION_ERROR` - Invalid input data (400)
- `INTERNAL_ERROR` - Server error (500)
- `CATEGORY_ALREADY_EXISTS` - Duplicate category name in same parent (409)
- `PRODUCT_NOT_FOUND` - Product not found (404)

---

## Notes

1. All pagination endpoints support `limit` (default 50, max 200) and `offset` (default 0)
2. Search parameter `q` searches across multiple fields (varies by module)
3. Soft deletes set `is_active=false` instead of removing records
4. List endpoints only show active records by default
5. Category tree endpoint returns hierarchical structure
6. Product card endpoint includes product + stock + categories in single response
