-- Backfill payroll compensation from employees.salary (UI source of truth).
-- Only rows with salary > 0; keep existing pay_day when present.
INSERT INTO worker_compensation (worker_id, base_salary_cents, pay_day, is_active)
SELECT
    e.id,
    ROUND(e.salary * 100)::bigint,
    1,
    true
FROM employees e
WHERE e.deleted_at IS NULL
  AND e.salary > 0
ON CONFLICT (worker_id) DO UPDATE
SET base_salary_cents = EXCLUDED.base_salary_cents,
    is_active = true,
    updated_at = now();
