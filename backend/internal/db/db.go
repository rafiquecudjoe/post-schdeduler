package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/scheduler/backend/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the database connection pool
type DB struct {
	pool *pgxpool.Pool
}

// PostWithRetry is an alias for models.Post used in worker retry logic
type PostWithRetry = models.Post

// New creates a new database connection
func New(ctx context.Context, databaseURL string) (*DB, error) {
	// Parse config to set optimal pool settings
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}
	
	// Optimize connection pool for low latency
	config.MaxConns = 25                          // Increase max connections
	config.MinConns = 5                           // Keep minimum connections ready
	config.MaxConnLifetime = 30 * time.Minute     // Connection lifetime
	config.MaxConnIdleTime = 5 * time.Minute      // Idle connection timeout
	config.HealthCheckPeriod = 30 * time.Second   // Health check interval
	
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.pool.Close()
}

// RunMigrations runs all database migrations
func (db *DB) RunMigrations(ctx context.Context) error {
	// Create migrations table if not exists
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort and filter for .up.sql files
	var upMigrations []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			upMigrations = append(upMigrations, entry.Name())
		}
	}
	sort.Strings(upMigrations)

	// Run each migration
	for _, filename := range upMigrations {
		version := strings.TrimSuffix(filename, ".up.sql")

		// Check if already applied
		var exists bool
		err := db.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}
		if exists {
			continue
		}

		// Read and execute migration
		content, err := fs.ReadFile(migrationsFS, "migrations/"+filename)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		_, err = db.pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", filename, err)
		}

		// Mark as applied
		_, err = db.pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		fmt.Printf("Applied migration: %s\n", filename)
	}

	return nil
}

// User operations

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Post operations

// CreatePost creates a new scheduled post
func (db *DB) CreatePost(ctx context.Context, userID uuid.UUID, title *string, content string, channel models.Channel, scheduledAt time.Time) (*models.Post, error) {
	post := &models.Post{}
	err := db.pool.QueryRow(ctx, `
		INSERT INTO posts (user_id, title, content, channel, scheduled_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
	`, userID, title, content, channel, scheduledAt).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
		&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return post, nil
}

// GetPostByID retrieves a post by ID
func (db *DB) GetPostByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.pool.QueryRow(ctx, `
		SELECT id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
		FROM posts WHERE id = $1
	`, id).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
		&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return post, nil
}

// GetUpcomingPosts retrieves scheduled posts for a user
func (db *DB) GetUpcomingPosts(ctx context.Context, userID uuid.UUID) ([]*models.Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
		FROM posts 
		WHERE user_id = $1 AND status = 'scheduled'
		ORDER BY scheduled_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		post := &models.Post{}
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
			&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// GetPublishedPosts retrieves published posts for a user
func (db *DB) GetPublishedPosts(ctx context.Context, userID uuid.UUID) ([]*models.Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
		FROM posts 
		WHERE user_id = $1 AND status = 'published'
		ORDER BY published_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		post := &models.Post{}
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
			&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// UpdatePost updates a scheduled post
func (db *DB) UpdatePost(ctx context.Context, id uuid.UUID, userID uuid.UUID, title *string, content *string, channel *models.Channel, scheduledAt *time.Time) (*models.Post, error) {
	// Only update fields that are provided
	post := &models.Post{}
	err := db.pool.QueryRow(ctx, `
		UPDATE posts SET
			title = COALESCE($3, title),
			content = COALESCE($4, content),
			channel = COALESCE($5, channel),
			scheduled_at = COALESCE($6, scheduled_at),
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'scheduled'
		RETURNING id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
	`, id, userID, title, content, channel, scheduledAt).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
		&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return post, nil
}

// DeletePost deletes a scheduled post
func (db *DB) DeletePost(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	result, err := db.pool.Exec(ctx, `
		DELETE FROM posts WHERE id = $1 AND user_id = $2 AND status = 'scheduled'
	`, id, userID)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}

// PublishPost marks a post as published (used by worker)
func (db *DB) PublishPost(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.pool.QueryRow(ctx, `
		UPDATE posts SET
			status = 'published',
			published_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND status = 'scheduled'
		RETURNING id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
	`, id).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
		&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return post, nil
}

// MarkPostFailed marks a post as failed with an error message
func (db *DB) MarkPostFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE posts SET 
			status = 'failed', 
			last_error = $2,
			updated_at = NOW()
		WHERE id = $1
	`, id, errorMsg)
	return err
}

// ScheduleRetry schedules a post for retry with exponential backoff
func (db *DB) ScheduleRetry(ctx context.Context, id uuid.UUID, nextRetryAt time.Time, errorMsg string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE posts SET 
			retry_count = retry_count + 1,
			last_error = $2,
			next_retry_at = $3,
			updated_at = NOW()
		WHERE id = $1 AND status = 'scheduled'
	`, id, errorMsg, nextRetryAt)
	return err
}

// GetPostForRetry retrieves a post with retry info for the worker
func (db *DB) GetPostForRetry(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	post := &models.Post{}
	err := db.pool.QueryRow(ctx, `
		SELECT id, user_id, title, content, channel, status, scheduled_at, published_at, 
			   retry_count, last_error, next_retry_at, created_at, updated_at
		FROM posts WHERE id = $1
	`, id).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
		&post.Status, &post.ScheduledAt, &post.PublishedAt,
		&post.RetryCount, &post.LastError, &post.NextRetryAt,
		&post.CreatedAt, &post.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return post, nil
}

// GetDuePosts retrieves posts that are due for publishing (for worker without Redis)
func (db *DB) GetDuePosts(ctx context.Context, limit int) ([]*models.Post, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, title, content, channel, status, scheduled_at, published_at, created_at, updated_at
		FROM posts 
		WHERE status = 'scheduled' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		post := &models.Post{}
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Title, &post.Content, &post.Channel,
			&post.Status, &post.ScheduledAt, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}
