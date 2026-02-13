-- Таблица для публичных ключей пользователей
CREATE TABLE IF NOT EXISTS user_public_keys (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    key_fingerprint VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Добавить поля в таблицу messages для шифрования
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS encryption_iv VARCHAR(255),
ADD COLUMN IF NOT EXISTS encryption_ver INTEGER DEFAULT 0;

-- Пометить все существующие сообщения как plaintext (версия 0)
UPDATE messages SET encryption_ver = 0 WHERE encryption_ver IS NULL;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_messages_encryption ON messages(encryption_ver);
