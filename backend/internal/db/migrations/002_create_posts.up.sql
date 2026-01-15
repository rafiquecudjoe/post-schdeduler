-- Create post status enum
CREATE TYPE post_status AS ENUM ('scheduled', 'published', 'failed');

-- Create channel type enum
CREATE TYPE channel_type AS ENUM ('twitter', 'linkedin', 'facebook');

-- Create posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    content TEXT NOT NULL,
    channel channel_type NOT NULL,
    status post_status DEFAULT 'scheduled',
    scheduled_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_user_scheduled ON posts(user_id, status, scheduled_at) WHERE status = 'scheduled';
CREATE INDEX idx_posts_user_published ON posts(user_id, status, published_at) WHERE status = 'published';
CREATE INDEX idx_posts_due ON posts(scheduled_at) WHERE status = 'scheduled';
