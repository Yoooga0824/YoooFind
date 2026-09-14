package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) RecordVisit(ctx context.Context, visitorKey string, userID int64) error {
	normalizedKey := strings.TrimSpace(visitorKey)
	if normalizedKey == "" {
		return nil
	}
	query := `
		INSERT IGNORE INTO visit_logs (visit_date, visitor_key, user_id)
		VALUES (CURRENT_DATE(), ?, NULLIF(?, 0))
	`
	if _, err := r.db.ExecContext(ctx, query, normalizedKey, userID); err != nil {
		return fmt.Errorf("record visit: %w", err)
	}
	return nil
}

func (r *AdminRepository) ListUsers(ctx context.Context) ([]model.AdminUserListItem, error) {
	query := `
		SELECT
			u.id,
			u.email,
			u.display_name,
			u.full_name,
			u.bio,
			u.avatar_path,
			u.daily_token_limit,
			DATE_FORMAT(u.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
			COALESCE(
				DATE_FORMAT(
					GREATEST(
						COALESCE(cs.session_last_at, '1970-01-01'),
						COALESCE(tu.usage_last_at, '1970-01-01'),
						COALESCE(u.updated_at, '1970-01-01')
					),
					'%Y-%m-%d %H:%i:%s'
				),
				''
			) AS last_active_at,
			COALESCE(tu.today_tokens, 0) AS today_tokens,
			COALESCE(tu.total_tokens, 0) AS total_tokens,
			COALESCE(cs.session_count, 0) AS session_count,
			COALESCE(cm.message_count, 0) AS message_count
		FROM users u
		LEFT JOIN (
			SELECT
				user_id,
				COALESCE(SUM(CASE WHEN DATE(created_at) = CURRENT_DATE() THEN total_tokens ELSE 0 END), 0) AS today_tokens,
				COALESCE(SUM(total_tokens), 0) AS total_tokens,
				MAX(created_at) AS usage_last_at
			FROM token_usage
			GROUP BY user_id
		) tu ON tu.user_id = u.id
		LEFT JOIN (
			SELECT
				user_id,
				COUNT(*) AS session_count,
				MAX(updated_at) AS session_last_at
			FROM chat_sessions
			GROUP BY user_id
		) cs ON cs.user_id = u.id
		LEFT JOIN (
			SELECT
				s.user_id,
				COUNT(m.id) AS message_count
			FROM chat_sessions s
			LEFT JOIN chat_messages m ON m.session_id = s.id
			GROUP BY s.user_id
		) cm ON cm.user_id = u.id
		ORDER BY u.created_at DESC, u.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	items := make([]model.AdminUserListItem, 0, 64)
	for rows.Next() {
		var item model.AdminUserListItem
		if err := rows.Scan(
			&item.ID,
			&item.Email,
			&item.DisplayName,
			&item.FullName,
			&item.Bio,
			&item.AvatarURL,
			&item.DailyTokenLimit,
			&item.CreatedAt,
			&item.LastActiveAt,
			&item.TodayTokens,
			&item.TotalTokens,
			&item.SessionCount,
			&item.MessageCount,
		); err != nil {
			return nil, fmt.Errorf("scan admin users: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin users: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) GetUserTokenSummary(ctx context.Context, userID int64) (model.AdminUserTokenSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURRENT_DATE() THEN total_tokens ELSE 0 END), 0) AS today_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		FROM token_usage
		WHERE user_id = ?
	`
	var summary model.AdminUserTokenSummary
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&summary.TodayTokens, &summary.TotalTokens); err != nil {
		return model.AdminUserTokenSummary{}, fmt.Errorf("query user token summary: %w", err)
	}
	return summary, nil
}

func (r *AdminRepository) GetUserTokenByDay(ctx context.Context, userID int64) ([]model.UsagePoint, error) {
	query := `
		SELECT
			DATE(created_at) AS d,
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(total_tokens), 0)
		FROM token_usage
		WHERE user_id = ?
		GROUP BY DATE(created_at)
		ORDER BY d ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user token by day: %w", err)
	}
	defer rows.Close()

	points := make([]model.UsagePoint, 0, 64)
	for rows.Next() {
		var point model.UsagePoint
		if err := rows.Scan(&point.Date, &point.PromptTokens, &point.CompletionTokens, &point.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan user token by day: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user token by day: %w", err)
	}
	return points, nil
}

func (r *AdminRepository) ListUserSessions(ctx context.Context, userID int64) ([]model.AdminChatSessionItem, error) {
	query := `
		SELECT
			id,
			title,
			DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		FROM chat_sessions
		WHERE user_id = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT 30
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]model.AdminChatSessionItem, 0, 32)
	for rows.Next() {
		var item model.AdminChatSessionItem
		if err := rows.Scan(&item.ID, &item.Title, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user sessions: %w", err)
		}
		sessions = append(sessions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user sessions: %w", err)
	}
	return sessions, nil
}

func (r *AdminRepository) ListSessionMessages(ctx context.Context, sessionID int64) ([]model.AdminChatMessageRow, error) {
	query := `
		SELECT
			role,
			content,
			reasoning_content,
			DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		FROM chat_messages
		WHERE session_id = ?
		ORDER BY created_at ASC, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list session messages: %w", err)
	}
	defer rows.Close()

	messages := make([]model.AdminChatMessageRow, 0, 128)
	for rows.Next() {
		var row model.AdminChatMessageRow
		if err := rows.Scan(&row.Role, &row.Content, &row.ReasoningContent, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session messages: %w", err)
		}
		if strings.TrimSpace(row.Role) == "assistant" {
			if payload, ok := model.ParsePersistedAssistantContent(row.Content); ok {
				selected := pickAdminSelectedModelResponse(payload.SelectedModel, payload.Responses)
				row.Content = selected.Content
				row.ReasoningContent = selected.ReasoningContent
				row.Model = selected.Model
			}
		}
		messages = append(messages, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session messages: %w", err)
	}
	return messages, nil
}

func (r *AdminRepository) GetVisitStats(ctx context.Context) (model.AdminVisitStats, error) {
	var stats model.AdminVisitStats

	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT visitor_key) FROM visit_logs`).Scan(&stats.TotalUniqueVisitors); err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("query total unique visitors: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT visitor_key) FROM visit_logs WHERE visit_date = CURRENT_DATE()`).Scan(&stats.TodayUniqueVisitors); err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("query today unique visitors: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT visitor_key) FROM visit_logs WHERE visitor_key LIKE 'user:%'`).Scan(&stats.LoggedInVisitors); err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("query logged in visitors: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT visitor_key) FROM visit_logs WHERE visitor_key LIKE 'anon:%'`).Scan(&stats.AnonymousVisitors); err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("query anonymous visitors: %w", err)
	}

	trendQuery := `
		SELECT visit_date, COUNT(DISTINCT visitor_key)
		FROM visit_logs
		WHERE visit_date >= CURRENT_DATE() - INTERVAL 29 DAY
		GROUP BY visit_date
		ORDER BY visit_date ASC
	`
	rows, err := r.db.QueryContext(ctx, trendQuery)
	if err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("query visit trend: %w", err)
	}
	defer rows.Close()

	trend := make([]model.AdminVisitPoint, 0, 30)
	for rows.Next() {
		var item model.AdminVisitPoint
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			return model.AdminVisitStats{}, fmt.Errorf("scan visit trend: %w", err)
		}
		trend = append(trend, item)
	}
	if err := rows.Err(); err != nil {
		return model.AdminVisitStats{}, fmt.Errorf("iterate visit trend: %w", err)
	}
	stats.DailyTrend = trend
	return stats, nil
}

func (r *AdminRepository) GetTokenOverview(ctx context.Context) (model.AdminTokenOverview, error) {
	var overview model.AdminTokenOverview
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE DATE(created_at) = CURRENT_DATE()`).Scan(&overview.TodayTotalTokens); err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("query today's token usage: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage`).Scan(&overview.HistoryTotal); err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("query history token usage: %w", err)
	}

	dailyQuery := `
		SELECT
			DATE(created_at) AS d,
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(total_tokens), 0)
		FROM token_usage
		WHERE DATE(created_at) >= CURRENT_DATE() - INTERVAL 29 DAY
		GROUP BY DATE(created_at)
		ORDER BY d ASC
	`
	dailyRows, err := r.db.QueryContext(ctx, dailyQuery)
	if err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("query daily token usage: %w", err)
	}
	defer dailyRows.Close()

	dailyTotal := make([]model.UsagePoint, 0, 30)
	for dailyRows.Next() {
		var item model.UsagePoint
		if err := dailyRows.Scan(&item.Date, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens); err != nil {
			return model.AdminTokenOverview{}, fmt.Errorf("scan daily token usage: %w", err)
		}
		dailyTotal = append(dailyTotal, item)
	}
	if err := dailyRows.Err(); err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("iterate daily token usage: %w", err)
	}
	overview.DailyTotal = dailyTotal

	usersQuery := `
		SELECT
			u.id,
			u.email,
			u.display_name,
			COALESCE(SUM(CASE WHEN DATE(t.created_at) = CURRENT_DATE() THEN t.total_tokens ELSE 0 END), 0) AS today_tokens,
			COALESCE(SUM(t.total_tokens), 0) AS total_tokens
		FROM users u
		LEFT JOIN token_usage t ON t.user_id = u.id
		GROUP BY u.id, u.email, u.display_name
		ORDER BY total_tokens DESC, u.id ASC
	`
	userRows, err := r.db.QueryContext(ctx, usersQuery)
	if err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("query per-user token summary: %w", err)
	}
	defer userRows.Close()

	users := make([]model.AdminUserTokenSummaryItem, 0, 64)
	for userRows.Next() {
		var item model.AdminUserTokenSummaryItem
		if err := userRows.Scan(&item.UserID, &item.Email, &item.DisplayName, &item.TodayTokens, &item.TotalTokens); err != nil {
			return model.AdminTokenOverview{}, fmt.Errorf("scan per-user token summary: %w", err)
		}
		users = append(users, item)
	}
	if err := userRows.Err(); err != nil {
		return model.AdminTokenOverview{}, fmt.Errorf("iterate per-user token summary: %w", err)
	}
	overview.Users = users

	return overview, nil
}

func pickAdminSelectedModelResponse(
	selectedModel string,
	responses []model.ModelAssistantResponse,
) model.ModelAssistantResponse {
	if len(responses) == 0 {
		return model.ModelAssistantResponse{}
	}
	selectedKey := strings.ToLower(strings.TrimSpace(selectedModel))
	for _, item := range responses {
		if strings.ToLower(strings.TrimSpace(item.Model)) == selectedKey {
			return item
		}
	}
	return responses[0]
}
