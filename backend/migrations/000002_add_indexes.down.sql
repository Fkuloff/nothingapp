-- Migration rollback: Remove indexes added in 000002_add_indexes.up.sql
-- Created: 2026-02-11

-- Remove text search indexes
DROP INDEX IF EXISTS idx_users_name_trgm;
DROP INDEX IF EXISTS idx_users_username_trgm;

-- Remove contact unique index
DROP INDEX IF EXISTS idx_contacts_user_contact;

-- Remove message pagination index
DROP INDEX IF EXISTS idx_messages_chat_created;

-- Remove unread message indexes
DROP INDEX IF EXISTS idx_unread_user_chat;
DROP INDEX IF EXISTS idx_unread_user_message;

-- Note: We don't drop pg_trgm extension as it might be used elsewhere
