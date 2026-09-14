package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"gemini-clone/backend/internal/model"
)

type Generator interface {
	GenerateReply(
		ctx context.Context,
		userMessage string,
		replyOptions model.ReplyOptions,
	) (model.AssistantReply, error)
	StreamReply(
		ctx context.Context,
		userMessage string,
		replyOptions model.ReplyOptions,
		onDelta func(model.AssistantReplyDelta) error,
	) (model.AssistantReply, error)
}

type ChatStore interface {
	CreateSession(ctx context.Context, userID int64, title string) (model.ChatSessionSummary, error)
	GetSession(ctx context.Context, userID, sessionID int64) (model.ChatSessionSummary, error)
	DeleteSession(ctx context.Context, userID, sessionID int64) error
	ListSessions(ctx context.Context, userID int64, limit int) ([]model.ChatSessionSummary, error)
	ListSessionMessages(ctx context.Context, userID, sessionID int64) ([]model.ChatMessageItem, error)
	ListRecentTurns(ctx context.Context, userID, sessionID int64, limit int) ([]model.ChatTurn, error)
	SaveTurn(ctx context.Context, userID, sessionID int64, userMessage, assistantContent, assistantReasoning string) error
	SaveUserMessage(ctx context.Context, userID, sessionID int64, userMessage string) error
	SaveAssistantMessage(ctx context.Context, userID, sessionID int64, assistantContent, assistantReasoning string) error
	UpdateSessionTitle(ctx context.Context, userID, sessionID int64, title string) error
}

type ChatService struct {
	generators   map[string]Generator
	modelOrder   []string
	store        ChatStore
	usageService *UsageService
}

type modelRunResult struct {
	Index int
	Item  model.ModelAssistantResponse
	Err   error
}

func NewChatService(
	generators map[string]Generator,
	modelOrder []string,
	store ChatStore,
	usageService *UsageService,
) *ChatService {
	normalizedOrder := make([]string, 0, len(modelOrder))
	seen := map[string]struct{}{}
	for _, item := range modelOrder {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := generators[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalizedOrder = append(normalizedOrder, key)
	}
	return &ChatService{
		generators:   generators,
		modelOrder:   normalizedOrder,
		store:        store,
		usageService: usageService,
	}
}

func (s *ChatService) Reply(
	ctx context.Context,
	userID int64,
	sessionID int64,
	userMessage string,
	replyOptions model.ReplyOptions,
) (model.AssistantReply, model.ChatSessionSummary, error) {
	replies, session, err := s.ReplyMulti(ctx, userID, sessionID, userMessage, nil, replyOptions)
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}
	if len(replies) == 0 {
		return model.AssistantReply{}, model.ChatSessionSummary{}, fmt.Errorf("empty model response")
	}
	return model.AssistantReply{
		Content:          replies[0].Content,
		ReasoningContent: replies[0].ReasoningContent,
		Usage:            replies[0].Usage,
	}, session, nil
}

func (s *ChatService) ReplyMulti(
	ctx context.Context,
	userID int64,
	sessionID int64,
	userMessage string,
	requestedModels []string,
	replyOptions model.ReplyOptions,
) ([]model.ModelAssistantResponse, model.ChatSessionSummary, error) {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return nil, model.ChatSessionSummary{}, fmt.Errorf("message cannot be empty")
	}
	if userID <= 0 {
		return nil, model.ChatSessionSummary{}, fmt.Errorf("user not authenticated")
	}
	if s.usageService != nil {
		if err := s.usageService.EnsureWithinDailyLimit(ctx, userID); err != nil {
			return nil, model.ChatSessionSummary{}, err
		}
	}

	modelKeys, _, err := s.resolveRequestedModels(requestedModels)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}

	session, err := s.ensureSession(ctx, userID, sessionID, trimmed)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	contextMessage, err := s.buildMessageWithRecentTurns(ctx, userID, session.ID, trimmed)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}

	assistantReplies, err := s.collectModelReplies(
		ctx,
		contextMessage,
		modelKeys,
		replyOptions,
		false,
		nil,
		nil,
	)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	session, err = s.persistAssistantResponses(ctx, userID, session, trimmed, assistantReplies)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	return assistantReplies, session, nil
}

func (s *ChatService) StreamReply(
	ctx context.Context,
	userID int64,
	sessionID int64,
	userMessage string,
	streamModel string,
	replyOptions model.ReplyOptions,
	onDelta func(model.AssistantReplyDelta) error,
) (model.AssistantReply, model.ChatSessionSummary, error) {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return model.AssistantReply{}, model.ChatSessionSummary{}, fmt.Errorf("message cannot be empty")
	}
	if userID <= 0 {
		return model.AssistantReply{}, model.ChatSessionSummary{}, fmt.Errorf("user not authenticated")
	}
	if s.usageService != nil {
		if err := s.usageService.EnsureWithinDailyLimit(ctx, userID); err != nil {
			return model.AssistantReply{}, model.ChatSessionSummary{}, err
		}
	}

	session, err := s.ensureSession(ctx, userID, sessionID, trimmed)
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}
	contextMessage, err := s.buildMessageWithRecentTurns(ctx, userID, session.ID, trimmed)
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}

	modelKey, err := s.normalizeSingleModel(streamModel)
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}
	generator, ok := s.generators[modelKey]
	if !ok {
		return model.AssistantReply{}, model.ChatSessionSummary{}, fmt.Errorf("model %s is unavailable", modelKey)
	}
	reply, err := generator.StreamReply(ctx, contextMessage, replyOptions, onDelta)
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}
	usage := ensureTokenUsage(reply.Usage, contextMessage, reply)
	session, err = s.persistAssistantResponses(ctx, userID, session, trimmed, []model.ModelAssistantResponse{
		{
			Model:            modelKey,
			Content:          reply.Content,
			ReasoningContent: reply.ReasoningContent,
			Usage:            usage,
		},
	})
	if err != nil {
		return model.AssistantReply{}, model.ChatSessionSummary{}, err
	}
	return reply, session, nil
}

func (s *ChatService) StreamReplyMulti(
	ctx context.Context,
	userID int64,
	sessionID int64,
	userMessage string,
	requestedModels []string,
	replyOptions model.ReplyOptions,
	onSessionReady func(model.ChatSessionSummary) error,
	onDelta func(modelKey string, delta model.AssistantReplyDelta) error,
	onModelError func(modelKey string, err error) error,
) ([]model.ModelAssistantResponse, model.ChatSessionSummary, error) {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return nil, model.ChatSessionSummary{}, fmt.Errorf("message cannot be empty")
	}
	if userID <= 0 {
		return nil, model.ChatSessionSummary{}, fmt.Errorf("user not authenticated")
	}
	if s.usageService != nil {
		if err := s.usageService.EnsureWithinDailyLimit(ctx, userID); err != nil {
			return nil, model.ChatSessionSummary{}, err
		}
	}

	modelKeys, unsupportedModels, err := s.resolveRequestedModels(requestedModels)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	if onModelError != nil {
		for _, modelKey := range unsupportedModels {
			_ = onModelError(modelKey, fmt.Errorf("model %s is unavailable", modelKey))
		}
	}
	session, err := s.ensureSession(ctx, userID, sessionID, trimmed)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	contextMessage, err := s.buildMessageWithRecentTurns(ctx, userID, session.ID, trimmed)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	if err := s.store.SaveUserMessage(ctx, userID, session.ID, trimmed); err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	if err := s.syncSessionTitle(ctx, userID, session, trimmed); err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	if onSessionReady != nil {
		if err := onSessionReady(session); err != nil {
			return nil, model.ChatSessionSummary{}, err
		}
	}

	assistantReplies, err := s.collectModelReplies(
		ctx,
		contextMessage,
		modelKeys,
		replyOptions,
		true,
		onDelta,
		onModelError,
	)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	session, err = s.persistStreamAssistantResponses(ctx, userID, session, assistantReplies)
	if err != nil {
		return nil, model.ChatSessionSummary{}, err
	}
	return assistantReplies, session, nil
}

func (s *ChatService) collectModelReplies(
	ctx context.Context,
	contextMessage string,
	modelKeys []string,
	replyOptions model.ReplyOptions,
	useStream bool,
	onDelta func(modelKey string, delta model.AssistantReplyDelta) error,
	onModelError func(modelKey string, err error) error,
) ([]model.ModelAssistantResponse, error) {
	// 并发执行模型请求并按前端选择顺序归并结果。
	resultsChan := make(chan modelRunResult, len(modelKeys))
	var wg sync.WaitGroup

	for idx, modelKey := range modelKeys {
		wg.Add(1)
		go func(i int, key string) {
			defer wg.Done()
			generator, ok := s.generators[key]
			if !ok {
				resultsChan <- modelRunResult{
					Index: i,
					Err:   fmt.Errorf("model %s is unavailable", key),
				}
				return
			}

			var (
				reply  model.AssistantReply
				runErr error
			)
			if useStream {
				reply, runErr = generator.StreamReply(
					ctx,
					contextMessage,
					replyOptions,
					func(delta model.AssistantReplyDelta) error {
						if onDelta == nil {
							return nil
						}
						return onDelta(key, delta)
					},
				)
			} else {
				reply, runErr = generator.GenerateReply(ctx, contextMessage, replyOptions)
			}
			if runErr != nil {
				if onModelError != nil {
					_ = onModelError(key, runErr)
				}
				resultsChan <- modelRunResult{
					Index: i,
					Err:   fmt.Errorf("%s: %w", key, runErr),
				}
				return
			}

			resultsChan <- modelRunResult{
				Index: i,
				Item: model.ModelAssistantResponse{
					Model:            key,
					Content:          reply.Content,
					ReasoningContent: reply.ReasoningContent,
					Usage:            ensureTokenUsage(reply.Usage, contextMessage, reply),
				},
			}
		}(idx, modelKey)
	}

	wg.Wait()
	close(resultsChan)

	successes := make([]modelRunResult, 0, len(modelKeys))
	errorsByModel := make([]string, 0)
	for result := range resultsChan {
		if result.Err != nil {
			errorsByModel = append(errorsByModel, result.Err.Error())
			continue
		}
		successes = append(successes, result)
	}
	sort.Slice(successes, func(i, j int) bool {
		return successes[i].Index < successes[j].Index
	})
	if len(successes) == 0 {
		return nil, fmt.Errorf("all model requests failed: %s", strings.Join(errorsByModel, " | "))
	}

	assistantReplies := make([]model.ModelAssistantResponse, 0, len(successes))
	for _, item := range successes {
		assistantReplies = append(assistantReplies, item.Item)
	}
	return assistantReplies, nil
}

func (s *ChatService) persistAssistantResponses(
	ctx context.Context,
	userID int64,
	session model.ChatSessionSummary,
	userMessage string,
	assistantReplies []model.ModelAssistantResponse,
) (model.ChatSessionSummary, error) {
	return s.persistAssistantResponsesWithMode(ctx, userID, session, userMessage, assistantReplies, false)
}

func (s *ChatService) persistStreamAssistantResponses(
	ctx context.Context,
	userID int64,
	session model.ChatSessionSummary,
	assistantReplies []model.ModelAssistantResponse,
) (model.ChatSessionSummary, error) {
	return s.persistAssistantResponsesWithMode(ctx, userID, session, "", assistantReplies, true)
}

func (s *ChatService) persistAssistantResponsesWithMode(
	ctx context.Context,
	userID int64,
	session model.ChatSessionSummary,
	userMessage string,
	assistantReplies []model.ModelAssistantResponse,
	userAlreadySaved bool,
) (model.ChatSessionSummary, error) {
	// 统一处理回复落库、标题同步、会话刷新与 token 统计，保证单/多模型路径一致。
	if len(assistantReplies) == 0 {
		return model.ChatSessionSummary{}, fmt.Errorf("empty model response")
	}

	primary := assistantReplies[0]
	assistantContent := primary.Content
	assistantReasoning := primary.ReasoningContent
	if len(assistantReplies) > 1 {
		payloadText, marshalErr := model.BuildPersistedAssistantContent(primary.Model, assistantReplies)
		if marshalErr != nil {
			return model.ChatSessionSummary{}, fmt.Errorf("encode multi model payload: %w", marshalErr)
		}
		assistantContent = payloadText
		assistantReasoning = ""
	}
	if userAlreadySaved {
		if err := s.store.SaveAssistantMessage(ctx, userID, session.ID, assistantContent, assistantReasoning); err != nil {
			return model.ChatSessionSummary{}, err
		}
	} else {
		if err := s.store.SaveTurn(ctx, userID, session.ID, userMessage, assistantContent, assistantReasoning); err != nil {
			return model.ChatSessionSummary{}, err
		}
	}
	if !userAlreadySaved {
		if err := s.syncSessionTitle(ctx, userID, session, userMessage); err != nil {
			return model.ChatSessionSummary{}, err
		}
	}
	updatedSession, err := s.store.GetSession(ctx, userID, session.ID)
	if err != nil {
		return model.ChatSessionSummary{}, err
	}
	s.recordUsage(ctx, userID, assistantReplies)
	return updatedSession, nil
}

func (s *ChatService) recordUsage(
	ctx context.Context,
	userID int64,
	replies []model.ModelAssistantResponse,
) {
	if s.usageService == nil {
		return
	}
	for _, item := range replies {
		if item.Usage == nil || item.Usage.TotalTokens <= 0 {
			continue
		}
		_ = s.usageService.RecordChatUsage(ctx, userID, item.Usage, item.Model)
	}
}

func (s *ChatService) normalizeSingleModel(input string) (string, error) {
	models, err := s.normalizeRequestedModels([]string{input})
	if err != nil {
		return "", err
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no model selected")
	}
	return models[0], nil
}

func (s *ChatService) normalizeRequestedModels(input []string) ([]string, error) {
	const maxModelsPerRequest = 4
	seen := map[string]struct{}{}
	models := make([]string, 0, maxModelsPerRequest)
	for _, item := range input {
		key := strings.TrimSpace(strings.ToLower(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := s.generators[key]; !ok {
			return nil, fmt.Errorf("unsupported model: %s", key)
		}
		seen[key] = struct{}{}
		models = append(models, key)
		if len(models) >= maxModelsPerRequest {
			break
		}
	}
	if len(models) > 0 {
		return models, nil
	}
	for _, key := range s.modelOrder {
		if _, ok := s.generators[key]; ok {
			return []string{key}, nil
		}
	}
	for key := range s.generators {
		return []string{key}, nil
	}
	return nil, fmt.Errorf("no models available")
}

func (s *ChatService) resolveRequestedModels(input []string) ([]string, []string, error) {
	const maxModelsPerRequest = 4
	seen := map[string]struct{}{}
	supported := make([]string, 0, maxModelsPerRequest)
	unsupported := make([]string, 0)

	for _, item := range input {
		key := strings.TrimSpace(strings.ToLower(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := s.generators[key]; ok {
			supported = append(supported, key)
		} else {
			unsupported = append(unsupported, key)
		}
		if len(supported)+len(unsupported) >= maxModelsPerRequest {
			break
		}
	}

	if len(supported) > 0 {
		return supported, unsupported, nil
	}
	if len(input) > 0 {
		return nil, unsupported, fmt.Errorf("no models available in selection")
	}
	defaultModels, err := s.normalizeRequestedModels(nil)
	return defaultModels, nil, err
}

func (s *ChatService) ListSessions(ctx context.Context, userID int64) ([]model.ChatSessionSummary, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user not authenticated")
	}
	return s.store.ListSessions(ctx, userID, 30)
}

func (s *ChatService) GetSessionDetail(
	ctx context.Context,
	userID int64,
	sessionID int64,
) (model.ChatSessionSummary, []model.ChatMessageItem, error) {
	if userID <= 0 {
		return model.ChatSessionSummary{}, nil, fmt.Errorf("user not authenticated")
	}
	session, err := s.store.GetSession(ctx, userID, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.ChatSessionSummary{}, nil, fmt.Errorf("session not found")
		}
		return model.ChatSessionSummary{}, nil, err
	}
	messages, err := s.store.ListSessionMessages(ctx, userID, sessionID)
	if err != nil {
		return model.ChatSessionSummary{}, nil, err
	}
	return session, messages, nil
}

func (s *ChatService) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	if userID <= 0 {
		return fmt.Errorf("user not authenticated")
	}
	if sessionID <= 0 {
		return fmt.Errorf("invalid session id")
	}
	if err := s.store.DeleteSession(ctx, userID, sessionID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("session not found")
		}
		return err
	}
	return nil
}

func (s *ChatService) ensureSession(
	ctx context.Context,
	userID int64,
	sessionID int64,
	userMessage string,
) (model.ChatSessionSummary, error) {
	if sessionID > 0 {
		session, err := s.store.GetSession(ctx, userID, sessionID)
		if err != nil {
			if err == sql.ErrNoRows {
				return model.ChatSessionSummary{}, fmt.Errorf("session not found")
			}
			return model.ChatSessionSummary{}, err
		}
		return session, nil
	}
	return s.store.CreateSession(ctx, userID, buildSessionTitle(userMessage))
}

func (s *ChatService) buildMessageWithRecentTurns(
	ctx context.Context,
	userID int64,
	sessionID int64,
	currentMessage string,
) (string, error) {
	turns, err := s.store.ListRecentTurns(ctx, userID, sessionID, 5)
	if err != nil {
		return "", err
	}
	if len(turns) == 0 {
		return currentMessage, nil
	}

	var builder strings.Builder
	builder.WriteString("你会收到当前对话的最近历史，请仅将其作为上下文，不要机械复述。\n\n")
	builder.WriteString("[最近历史]\n")
	for _, turn := range turns {
		if strings.TrimSpace(turn.AssistantContent) == "" && strings.TrimSpace(turn.AssistantReasoning) == "" {
			continue
		}
		if turn.UserMessage != "" {
			builder.WriteString("用户: ")
			builder.WriteString(turn.UserMessage)
			builder.WriteString("\n")
		}
		if turn.AssistantContent != "" {
			builder.WriteString("助手: ")
			builder.WriteString(turn.AssistantContent)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n[当前问题]\n")
	builder.WriteString(currentMessage)
	return builder.String(), nil
}

func (s *ChatService) syncSessionTitle(
	ctx context.Context,
	userID int64,
	session model.ChatSessionSummary,
	userMessage string,
) error {
	if strings.TrimSpace(session.Title) != "新聊天" {
		return nil
	}
	return s.store.UpdateSessionTitle(ctx, userID, session.ID, buildSessionTitle(userMessage))
}

func buildSessionTitle(text string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if normalized == "" {
		return "新聊天"
	}
	runes := []rune(normalized)
	if len(runes) > 18 {
		return string(runes[:18]) + "..."
	}
	return normalized
}

func ensureTokenUsage(
	raw *model.TokenUsage,
	prompt string,
	reply model.AssistantReply,
) *model.TokenUsage {
	if raw != nil && raw.TotalTokens > 0 {
		return raw
	}

	promptTokens := estimateTokenCount(prompt)
	completionText := strings.TrimSpace(strings.Join([]string{
		reply.Content,
		reply.ReasoningContent,
	}, "\n"))
	completionTokens := estimateTokenCount(completionText)

	if promptTokens == 0 && completionTokens == 0 {
		return nil
	}

	return &model.TokenUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

func estimateTokenCount(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	runes := len([]rune(trimmed))
	approx := int(math.Ceil(float64(runes) / 4.0))
	if approx < 1 {
		return 1
	}
	return approx
}
