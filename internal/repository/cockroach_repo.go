package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"twitter/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CockroachRepo struct {
	pool *pgxpool.Pool
}

func NewCockroachRepo(pool *pgxpool.Pool) *CockroachRepo {
	return &CockroachRepo{pool: pool}
}

func (r *CockroachRepo) InsertMessage(ctx context.Context, content string) (*models.Message, error) {
	query := `
		INSERT INTO messages (content) 
		VALUES ($1) 
		RETURNING id, content, created_at;`

	var msg models.Message
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		err := r.pool.QueryRow(ctx, query, content).Scan(&msg.ID, &msg.Content, &msg.CreatedAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "40001" {
				continue
			}
			return nil, fmt.Errorf("failed to insert message: %w", err)
		}
		return &msg, nil
	}

	return nil, fmt.Errorf("max retries exceeded inserting message")
}

func (r *CockroachRepo) GetRecentMessages(ctx context.Context, limit int) ([]*models.Message, error) {
	query := `
		SELECT id, content, created_at 
		FROM messages 
		ORDER BY created_at DESC, id DESC 
		LIMIT $1;`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func (r *CockroachRepo) GetFeedCursor(ctx context.Context, limit int, cursorTime time.Time, cursorID string) ([]*models.Message, error) {
	query := `
		SELECT id, content, created_at 
		FROM messages 
		WHERE (created_at, id) < ($1, $2)
		ORDER BY created_at DESC, id DESC 
		LIMIT $3;`

	rows, err := r.pool.Query(ctx, query, cursorTime, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query cursor messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func scanMessages(rows pgx.Rows) ([]*models.Message, error) {
	var messages []*models.Message

	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}
