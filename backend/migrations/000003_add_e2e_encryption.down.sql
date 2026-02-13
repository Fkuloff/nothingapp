-- Откат миграции E2E шифрования
ALTER TABLE messages DROP COLUMN IF EXISTS encryption_iv;
ALTER TABLE messages DROP COLUMN IF EXISTS encryption_ver;
DROP INDEX IF EXISTS idx_messages_encryption;
DROP TABLE IF EXISTS user_public_keys;
