-- 001_types.sql

CREATE TYPE customer_type AS ENUM ('regular', 'vip', 'wholesale');

CREATE TYPE role_code AS ENUM ('admin', 'cashier', 'operator', 'manager');
