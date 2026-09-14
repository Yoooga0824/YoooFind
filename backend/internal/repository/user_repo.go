package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (int64, error) {
	query := `
		INSERT INTO users (email, password_hash, display_name, daily_token_limit)
		VALUES (?, ?, '用户', 1000000)
	`
	res, err := r.db.ExecContext(ctx, query, strings.ToLower(strings.TrimSpace(email)), passwordHash)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get user id: %w", err)
	}
	return id, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, full_name, bio, avatar_path, daily_token_limit
		FROM users
		WHERE email = ?
		LIMIT 1
	`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.FullName,
		&user.Bio,
		&user.AvatarPath,
		&user.DailyTokenLimit,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, sql.ErrNoRows
		}
		return model.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, full_name, bio, avatar_path, daily_token_limit
		FROM users
		WHERE id = ?
		LIMIT 1
	`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.FullName,
		&user.Bio,
		&user.AvatarPath,
		&user.DailyTokenLimit,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, sql.ErrNoRows
		}
		return model.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID int64, req model.UpdateProfileRequest) error {
	query := `
		UPDATE users
		SET display_name = ?, full_name = ?, bio = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		strings.TrimSpace(req.DisplayName),
		strings.TrimSpace(req.FullName),
		strings.TrimSpace(req.Bio),
		userID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateAvatarPath(ctx context.Context, userID int64, avatarPath string) error {
	query := `UPDATE users SET avatar_path = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, avatarPath, userID)
	if err != nil {
		return fmt.Errorf("update avatar: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateDailyTokenLimit(ctx context.Context, userID int64, dailyTokenLimit int64) error {
	query := `UPDATE users SET daily_token_limit = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, dailyTokenLimit, userID); err != nil {
		return fmt.Errorf("update daily token limit: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, passwordHash, userID); err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	return nil
}

func (r *UserRepository) ListAll(ctx context.Context) ([]model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, full_name, bio, avatar_path, daily_token_limit
		FROM users
		ORDER BY created_at DESC, id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0, 64)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.DisplayName,
			&user.FullName,
			&user.Bio,
			&user.AvatarPath,
			&user.DailyTokenLimit,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, nil
}
