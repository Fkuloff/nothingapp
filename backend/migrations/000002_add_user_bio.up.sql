-- Migration: add_user_bio
-- Add biography field to users table

ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN status VARCHAR(50) DEFAULT 'offline';

-- Create index for status
CREATE INDEX idx_users_status ON users(status);
