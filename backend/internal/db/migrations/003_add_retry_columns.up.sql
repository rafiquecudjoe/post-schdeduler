-- Add retry tracking columns to posts table
ALTER TABLE posts ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS last_error TEXT;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

-- Index for finding posts to retry
CREATE INDEX IF NOT EXISTS idx_posts_next_retry_at ON posts(next_retry_at) WHERE status = 'scheduled' AND next_retry_at IS NOT NULL;
