package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
)

type FeedbackRepository struct {
	db *sql.DB
}

func NewFeedbackRepository(db *sql.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(
	ctx context.Context,
	userID int64,
	title, content, userEmail, userDisplayName string,
) (int64, error) {
	query := `
		INSERT INTO feedback (user_id, title, content, user_email, user_display_name, status)
		VALUES (?, ?, ?, ?, ?, 'new')
	`
	var nullableUserID any
	if userID > 0 {
		nullableUserID = userID
	}
	res, err := r.db.ExecContext(
		ctx,
		query,
		nullableUserID,
		strings.TrimSpace(title),
		strings.TrimSpace(content),
		strings.TrimSpace(userEmail),
		strings.TrimSpace(userDisplayName),
	)
	if err != nil {
		return 0, fmt.Errorf("create feedback: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get feedback id: %w", err)
	}
	return id, nil
}

func (r *FeedbackRepository) ListAll(ctx context.Context) ([]model.FeedbackItem, error) {
	query := `
		SELECT id, COALESCE(user_id, 0), title, content, user_email, user_display_name, status, created_at
		FROM feedback
		ORDER BY created_at DESC, id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list feedback: %w", err)
	}
	defer rows.Close()

	items := make([]model.FeedbackItem, 0)
	for rows.Next() {
		var item model.FeedbackItem
		var createdAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Title,
			&item.Content,
			&item.UserEmail,
			&item.UserDisplayName,
			&item.Status,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan feedback row: %w", err)
		}
		if createdAt.Valid {
			item.CreatedAt = createdAt.Time.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feedback rows: %w", err)
	}
	return items, nil
}

func (r *FeedbackRepository) UpdateStatus(ctx context.Context, feedbackID int64, status string) error {
	query := `UPDATE feedback SET status = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, status, feedbackID)
	if err != nil {
		return fmt.Errorf("update feedback status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
