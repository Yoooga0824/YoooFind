package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
)

const (
	maxSessionsPerUser = 30
)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateSession(ctx context.Context, userID int64, title string) (model.ChatSessionSummary, error) {
	cleanTitle := normalizeSessionTitle(title)
	insertQuery := `
		INSERT INTO chat_sessions (user_id, title)
		VALUES (?, ?)
	`
	res, err := r.db.ExecContext(ctx, insertQuery, userID, cleanTitle)
	if err != nil {
		return model.ChatSessionSummary{}, fmt.Errorf("create chat session: %w", err)
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return model.ChatSessionSummary{}, fmt.Errorf("get chat session id: %w", err)
	}

	trimQuery := `
		DELETE s
		FROM chat_sessions s
		JOIN (
			SELECT id
			FROM chat_sessions
			WHERE user_id = ?
			ORDER BY updated_at DESC, id DESC
			LIMIT 18446744073709551615 OFFSET ?
		) old ON old.id = s.id
		WHERE s.user_id = ?
	`
	if _, err := r.db.ExecContext(ctx, trimQuery, userID, maxSessionsPerUser, userID); err != nil {
		return model.ChatSessionSummary{}, fmt.Errorf("trim old chat sessions: %w", err)
	}

	return r.GetSession(ctx, userID, sessionID)
}

func (r *ChatRepository) GetSession(ctx context.Context, userID, sessionID int64) (model.ChatSessionSummary, error) {
	query := `
		SELECT id, title, DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s')
		FROM chat_sessions
		WHERE id = ? AND user_id = ?
		LIMIT 1
	`
	var session model.ChatSessionSummary
	err := r.db.QueryRowContext(ctx, query, sessionID, userID).Scan(
		&session.ID,
		&session.Title,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ChatSessionSummary{}, sql.ErrNoRows
		}
		return model.ChatSessionSummary{}, fmt.Errorf("get chat session: %w", err)
	}
	return session, nil
}

func (r *ChatRepository) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	query := `
		DELETE FROM chat_sessions
		WHERE id = ? AND user_id = ?
	`
	result, err := r.db.ExecContext(ctx, query, sessionID, userID)
	if err != nil {
		return fmt.Errorf("delete chat session: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for delete chat session: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ChatRepository) ListSessions(ctx context.Context, userID int64, limit int) ([]model.ChatSessionSummary, error) {
	if limit <= 0 || limit > maxSessionsPerUser {
		limit = maxSessionsPerUser
	}
	query := `
		SELECT id, title, DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s')
		FROM chat_sessions
		WHERE user_id = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list chat sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]model.ChatSessionSummary, 0, limit)
	for rows.Next() {
		var session model.ChatSessionSummary
		if err := rows.Scan(&session.ID, &session.Title, &session.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan chat session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat sessions: %w", err)
	}

	return sessions, nil
}

func (r *ChatRepository) ListSessionMessages(ctx context.Context, userID, sessionID int64) ([]model.ChatMessageItem, error) {
	query := `
		SELECT m.role, m.content, m.reasoning_content, DATE_FORMAT(m.created_at, '%Y-%m-%d %H:%i:%s')
		FROM chat_messages m
		JOIN chat_sessions s ON s.id = m.session_id
		WHERE m.session_id = ? AND s.user_id = ?
		ORDER BY m.created_at ASC, m.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]model.ChatMessageItem, 0, 64)
	for rows.Next() {
		var msg model.ChatMessageItem
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.ReasoningContent, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		if strings.TrimSpace(msg.Role) == "assistant" {
			if payload, ok := model.ParsePersistedAssistantContent(msg.Content); ok {
				msg.SelectedModel = payload.SelectedModel
				msg.ModelResponses = payload.Responses
				selected := pickSelectedModelResponse(payload.SelectedModel, payload.Responses)
				msg.Content = selected.Content
				msg.ReasoningContent = selected.ReasoningContent
				msg.Model = selected.Model
			}
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat messages: %w", err)
	}

	return messages, nil
}

func (r *ChatRepository) ListRecentTurns(ctx context.Context, userID, sessionID int64, limit int) ([]model.ChatTurn, error) {
	if limit <= 0 {
		limit = 5
	}
	query := `
		SELECT
			m.role, m.content, m.reasoning_content
		FROM (
			SELECT m.id, m.created_at, m.role, m.content, m.reasoning_content
			FROM chat_messages m
			JOIN chat_sessions s ON s.id = m.session_id
			WHERE m.session_id = ? AND s.user_id = ?
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT ?
		) m
		ORDER BY m.created_at ASC, m.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID, userID, limit*2)
	if err != nil {
		return nil, fmt.Errorf("list recent turns: %w", err)
	}
	defer rows.Close()

	turns := make([]model.ChatTurn, 0, limit)
	currentTurn := model.ChatTurn{}
	hasCurrent := false
	for rows.Next() {
		var (
			role      string
			content   string
			reasoning string
		)
		if err := rows.Scan(&role, &content, &reasoning); err != nil {
			return nil, fmt.Errorf("scan recent message for turns: %w", err)
		}

		role = strings.TrimSpace(role)
		content = strings.TrimSpace(content)
		reasoning = strings.TrimSpace(reasoning)
		if role == "assistant" {
			if payload, ok := model.ParsePersistedAssistantContent(content); ok {
				selected := pickSelectedModelResponse(payload.SelectedModel, payload.Responses)
				content = strings.TrimSpace(selected.Content)
				reasoning = strings.TrimSpace(selected.ReasoningContent)
			}
		}
		if role == "user" {
			if hasCurrent {
				turns = append(turns, currentTurn)
				if len(turns) >= limit {
					break
				}
			}
			currentTurn = model.ChatTurn{UserMessage: content}
			hasCurrent = true
			continue
		}
		if role == "assistant" {
			if !hasCurrent {
				currentTurn = model.ChatTurn{}
				hasCurrent = true
			}
			currentTurn.AssistantContent = content
			currentTurn.AssistantReasoning = reasoning
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent turns: %w", err)
	}
	if hasCurrent && len(turns) < limit {
		turns = append(turns, currentTurn)
	}

	filtered := make([]model.ChatTurn, 0, len(turns))
	for _, t := range turns {
		if t.UserMessage == "" && t.AssistantContent == "" && t.AssistantReasoning == "" {
			continue
		}
		filtered = append(filtered, t)
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered, nil
}

func (r *ChatRepository) SaveUserMessage(
	ctx context.Context,
	userID, sessionID int64,
	userMessage string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save user message transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const ensureSessionQuery = `
		SELECT id
		FROM chat_sessions
		WHERE id = ? AND user_id = ?
		LIMIT 1
	`
	var ensuredID int64
	if err := tx.QueryRowContext(ctx, ensureSessionQuery, sessionID, userID).Scan(&ensuredID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("ensure chat session owner: %w", err)
	}

	insertQuery := `
		INSERT INTO chat_messages (session_id, role, content, reasoning_content)
		VALUES (?, 'user', ?, '')
	`
	if _, err := tx.ExecContext(
		ctx,
		insertQuery,
		sessionID,
		strings.TrimSpace(userMessage),
	); err != nil {
		return fmt.Errorf("save user chat message: %w", err)
	}

	updateQuery := `
		UPDATE chat_sessions
		SET updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`
	if _, err := tx.ExecContext(ctx, updateQuery, sessionID, userID); err != nil {
		return fmt.Errorf("touch chat session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save user message transaction: %w", err)
	}
	return nil
}

func (r *ChatRepository) SaveAssistantMessage(
	ctx context.Context,
	userID, sessionID int64,
	assistantContent, assistantReasoning string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save assistant message transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const ensureSessionQuery = `
		SELECT id
		FROM chat_sessions
		WHERE id = ? AND user_id = ?
		LIMIT 1
	`
	var ensuredID int64
	if err := tx.QueryRowContext(ctx, ensureSessionQuery, sessionID, userID).Scan(&ensuredID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("ensure chat session owner: %w", err)
	}

	insertQuery := `
		INSERT INTO chat_messages (session_id, role, content, reasoning_content)
		VALUES (?, 'assistant', ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		insertQuery,
		sessionID,
		strings.TrimSpace(assistantContent),
		strings.TrimSpace(assistantReasoning),
	); err != nil {
		return fmt.Errorf("save assistant chat message: %w", err)
	}

	updateQuery := `
		UPDATE chat_sessions
		SET updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`
	if _, err := tx.ExecContext(ctx, updateQuery, sessionID, userID); err != nil {
		return fmt.Errorf("touch chat session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save assistant message transaction: %w", err)
	}
	return nil
}

func (r *ChatRepository) SaveTurn(
	ctx context.Context,
	userID, sessionID int64,
	userMessage, assistantContent, assistantReasoning string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save turn transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const ensureSessionQuery = `
		SELECT id
		FROM chat_sessions
		WHERE id = ? AND user_id = ?
		LIMIT 1
	`
	var ensuredID int64
	if err := tx.QueryRowContext(ctx, ensureSessionQuery, sessionID, userID).Scan(&ensuredID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("ensure chat session owner: %w", err)
	}

	insertQuery := `
		INSERT INTO chat_messages (session_id, role, content, reasoning_content)
		VALUES (?, 'user', ?, ''), (?, 'assistant', ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		insertQuery,
		sessionID,
		strings.TrimSpace(userMessage),
		sessionID,
		strings.TrimSpace(assistantContent),
		strings.TrimSpace(assistantReasoning),
	); err != nil {
		return fmt.Errorf("save chat messages: %w", err)
	}

	updateQuery := `
		UPDATE chat_sessions
		SET updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`
	if _, err := tx.ExecContext(ctx, updateQuery, sessionID, userID); err != nil {
		return fmt.Errorf("touch chat session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save turn transaction: %w", err)
	}
	return nil
}

func (r *ChatRepository) UpdateSessionTitle(ctx context.Context, userID, sessionID int64, title string) error {
	query := `
		UPDATE chat_sessions
		SET title = ?
		WHERE id = ? AND user_id = ?
	`
	if _, err := r.db.ExecContext(ctx, query, normalizeSessionTitle(title), sessionID, userID); err != nil {
		return fmt.Errorf("update chat session title: %w", err)
	}
	return nil
}

func normalizeSessionTitle(input string) string {
	title := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if title == "" {
		return "新聊天"
	}
	runes := []rune(title)
	if len(runes) > 40 {
		return string(runes[:40]) + "..."
	}
	return title
}

func pickSelectedModelResponse(
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
