-- Remove retry tracking columns from posts table
ALTER TABLE posts DROP COLUMN IF EXISTS retry_count;
ALTER TABLE posts DROP COLUMN IF EXISTS last_error;
ALTER TABLE posts DROP COLUMN IF EXISTS next_retry_at;

DROP INDEX IF EXISTS idx_posts_next_retry_at;
