-- Remove performance indexes
DROP INDEX IF EXISTS idx_posts_user_published;
DROP INDEX IF EXISTS idx_posts_user_scheduled;
DROP INDEX IF EXISTS idx_posts_user_status;
