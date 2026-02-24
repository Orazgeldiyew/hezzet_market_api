-- 017_sms_logs.down.sql

DROP TRIGGER IF EXISTS sms_logs_set_updated_at ON sms_logs;
DROP FUNCTION IF EXISTS trg_sms_logs_updated_at();
DROP TABLE IF EXISTS sms_logs;
