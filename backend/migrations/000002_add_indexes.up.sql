-- Migration: Add critical database indexes for performance and data integrity
-- Created: 2026-02-11

-- 1. CRITICAL: Prevent duplicate unread messages
-- This ensures that each (user, message) pair appears only once in unread_messages
CREATE UNIQUE INDEX IF NOT EXISTS idx_unread_user_message
ON unread_messages(user_id, message_id);

-- 2. HIGH: Fast unread message deletion by (user_id, chat_id)
-- Used when marking chat as read: DELETE WHERE user_id = ? AND chat_id = ?
CREATE INDEX IF NOT EXISTS idx_unread_user_chat
ON unread_messages(user_id, chat_id);

-- 3. HIGH: Message pagination performance
-- Used for efficient message loading with ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_messages_chat_created
ON messages(chat_id, created_at DESC);

-- 4. MEDIUM: Prevent duplicate contacts
-- Ensures each user can only have one contact entry for another user
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_user_contact
ON contacts(user_id, contact_user_id);

-- 5. MEDIUM: Text search optimization for user search
-- Enable pg_trgm extension for fuzzy text search (ILIKE queries)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN index for username search (supports ILIKE '%search%')
CREATE INDEX IF NOT EXISTS idx_users_username_trgm
ON users USING gin(username gin_trgm_ops);

-- GIN index for name search (supports ILIKE '%search%')
CREATE INDEX IF NOT EXISTS idx_users_name_trgm
ON users USING gin(name gin_trgm_ops);
