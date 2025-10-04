-- Users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    nickname VARCHAR(100),
    avatar TEXT,
    bio TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Conversations table
CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL CHECK (type IN ('direct', 'group', 'channel')),
    title TEXT,
    owner_id BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_type ON conversations(type);
CREATE INDEX idx_conversations_owner ON conversations(owner_id);

-- Conversation members table
CREATE TABLE conversation_members (
    conv_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'publisher', 'member', 'viewer')),
    muted BOOLEAN NOT NULL DEFAULT false,
    last_read_seq BIGINT NOT NULL DEFAULT 0,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (conv_id, user_id)
);

CREATE INDEX idx_conv_members_user ON conversation_members(user_id);
CREATE INDEX idx_conv_members_conv ON conversation_members(conv_id);

-- Messages table (partitioned by created_at)
CREATE TABLE messages (
    conv_id BIGINT NOT NULL,
    seq BIGINT NOT NULL,
    msg_id UUID PRIMARY KEY,
    sender_id BIGINT NOT NULL,
    conv_type VARCHAR(20) NOT NULL CHECK (conv_type IN ('direct', 'group', 'channel')),
    body JSONB NOT NULL,
    reply_to UUID,
    mentions BIGINT[] DEFAULT '{}',
    visibility TEXT DEFAULT 'normal',
    created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create indexes on messages table
CREATE INDEX idx_messages_conv_seq ON messages (conv_id, seq DESC);
CREATE INDEX idx_messages_sender_time ON messages (sender_id, created_at DESC);
CREATE INDEX idx_messages_created_at ON messages (created_at);

-- Files table (partitioned by created_at)
CREATE TABLE files (
    id UUID PRIMARY KEY,
    filename TEXT NOT NULL,
    size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    sender_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_files_sender ON files (sender_id);
CREATE INDEX idx_files_created_at ON files (created_at);

-- Conversation sequence table (for generating message seq)
CREATE TABLE conversation_seq (
    conv_id BIGINT PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    current_seq BIGINT NOT NULL DEFAULT 0
);

-- Function to get next sequence for a conversation
CREATE OR REPLACE FUNCTION next_conv_seq(p_conv_id BIGINT)
RETURNS BIGINT AS $$
DECLARE
    v_seq BIGINT;
BEGIN
    INSERT INTO conversation_seq (conv_id, current_seq)
    VALUES (p_conv_id, 1)
    ON CONFLICT (conv_id)
    DO UPDATE SET current_seq = conversation_seq.current_seq + 1
    RETURNING current_seq INTO v_seq;

    RETURN v_seq;
END;
$$ LANGUAGE plpgsql;

-- Function to create partition for messages
CREATE OR REPLACE FUNCTION create_messages_partition(partition_date DATE)
RETURNS void AS $$
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    partition_name := 'messages_' || to_char(partition_date, 'YYYY_MM_DD');
    start_date := partition_date;
    end_date := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF messages
         FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Function to create partition for files
CREATE OR REPLACE FUNCTION create_files_partition(partition_date DATE)
RETURNS void AS $$
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    partition_name := 'files_' || to_char(partition_date, 'YYYY_MM_DD');
    start_date := partition_date;
    end_date := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF files
         FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Function to drop old partitions (older than 30 days)
CREATE OR REPLACE FUNCTION drop_old_partitions()
RETURNS void AS $$
DECLARE
    partition_record RECORD;
    cutoff_date DATE;
BEGIN
    cutoff_date := CURRENT_DATE - INTERVAL '30 days';

    -- Drop old message partitions
    FOR partition_record IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'messages_%'
    LOOP
        DECLARE
            partition_date DATE;
        BEGIN
            partition_date := to_date(substring(partition_record.tablename from 10), 'YYYY_MM_DD');
            IF partition_date < cutoff_date THEN
                EXECUTE 'DROP TABLE IF EXISTS ' || partition_record.tablename;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            CONTINUE;
        END;
    END LOOP;

    -- Drop old file partitions
    FOR partition_record IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'files_%'
    LOOP
        DECLARE
            partition_date DATE;
        BEGIN
            partition_date := to_date(substring(partition_record.tablename from 7), 'YYYY_MM_DD');
            IF partition_date < cutoff_date THEN
                EXECUTE 'DROP TABLE IF EXISTS ' || partition_record.tablename;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            CONTINUE;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create initial partitions for today and tomorrow
SELECT create_messages_partition(CURRENT_DATE);
SELECT create_messages_partition(CURRENT_DATE + INTERVAL '1 day');
SELECT create_files_partition(CURRENT_DATE);
SELECT create_files_partition(CURRENT_DATE + INTERVAL '1 day');
