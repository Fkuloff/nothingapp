-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) UNIQUE NOT NULL,
    avatar_url VARCHAR(500)
);

-- Create index for soft deletes
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Create chats table
CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user1_id BIGINT NOT NULL,
    user2_id BIGINT NOT NULL,
    CONSTRAINT fk_chats_user1 FOREIGN KEY (user1_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_chats_user2 FOREIGN KEY (user2_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create unique index for chat participants (normalized: user1_id < user2_id)
CREATE UNIQUE INDEX idx_chat_users ON chats(user1_id, user2_id) WHERE deleted_at IS NULL;

-- Create indexes for queries
CREATE INDEX idx_chats_deleted_at ON chats(deleted_at);
CREATE INDEX idx_user1 ON chats(user1_id);
CREATE INDEX idx_user2 ON chats(user2_id);

-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    text TEXT NOT NULL,
    reply_to_id BIGINT,
    edited_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN DEFAULT FALSE,
    CONSTRAINT fk_messages_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_reply_to FOREIGN KEY (reply_to_id) REFERENCES messages(id) ON DELETE SET NULL
);

-- Create indexes for messages
CREATE INDEX idx_messages_deleted_at ON messages(deleted_at);
CREATE INDEX idx_chat_messages ON messages(chat_id);
CREATE INDEX idx_user_messages ON messages(user_id);
CREATE INDEX idx_reply_to ON messages(reply_to_id);

-- Create contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id BIGINT NOT NULL,
    contact_user_id BIGINT NOT NULL,
    CONSTRAINT fk_contacts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_contacts_contact_user FOREIGN KEY (contact_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for contacts
CREATE INDEX idx_contacts_deleted_at ON contacts(deleted_at);
CREATE INDEX idx_contacts_user_id ON contacts(user_id);
CREATE INDEX idx_contacts_contact_user_id ON contacts(contact_user_id);

-- Create attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    message_id BIGINT NOT NULL,
    file_type VARCHAR(20) NOT NULL,
    storage_key VARCHAR(500) UNIQUE NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    thumbnail_key VARCHAR(500),
    width INTEGER,
    height INTEGER,
    duration INTEGER,
    CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- Create indexes for attachments
CREATE INDEX idx_attachments_deleted_at ON attachments(deleted_at);
CREATE INDEX idx_message_attachments ON attachments(message_id);
CREATE INDEX idx_file_type ON attachments(file_type);

