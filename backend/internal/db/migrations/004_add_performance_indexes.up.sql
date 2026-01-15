-- Add indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_posts_user_status ON posts(user_id, status);
CREATE INDEX IF NOT EXISTS idx_posts_user_scheduled ON posts(user_id, scheduled_at) WHERE status = 'scheduled';
CREATE INDEX IF NOT EXISTS idx_posts_user_published ON posts(user_id, published_at DESC) WHERE status = 'published';
