package model

// ChatRequest is what frontend sends to backend.
// Keep it minimal and easy for beginners.
type ChatRequest struct {
	Message    string   `json:"message"`
	SessionID  int64    `json:"session_id,omitempty"`
	Models     []string `json:"models,omitempty"`
	DeepSearch bool     `json:"deep_search,omitempty"`
}

// ErrorEnvelope provides a consistent error shape to frontend.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
}

// OpenAICompatibleResponse is intentionally shaped like:
// /v1/chat/completions basic response, so frontend can parse choices[0].message.content
type OpenAICompatibleResponse struct {
	Choices []Choice            `json:"choices"`
	Usage   *TokenUsage         `json:"usage,omitempty"`
	Session *ChatSessionSummary `json:"session,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	Model            string `json:"model,omitempty"`
}

type ModelAssistantResponse struct {
	Model            string      `json:"model"`
	Content          string      `json:"content"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
	Usage            *TokenUsage `json:"usage,omitempty"`
	Error            string      `json:"error,omitempty"`
}

// AssistantReply is an internal normalized shape from provider to service.
// Content and reasoning are separated so frontend can render them in different panels.
type AssistantReply struct {
	Content          string
	ReasoningContent string
	Usage            *TokenUsage
}

// ReplyOptions carries optional generation switches.
type ReplyOptions struct {
	DeepSearch bool
}

// AssistantReplyDelta is one streamed incremental chunk from upstream.
type AssistantReplyDelta struct {
	Content          string
	ReasoningContent string
}

type ChatSessionSummary struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ChatMessageItem struct {
	Role             string                   `json:"role"`
	Content          string                   `json:"content"`
	ReasoningContent string                   `json:"reasoning_content,omitempty"`
	Model            string                   `json:"model,omitempty"`
	SelectedModel    string                   `json:"selected_model,omitempty"`
	ModelResponses   []ModelAssistantResponse `json:"model_responses,omitempty"`
	CreatedAt        string                   `json:"created_at,omitempty"`
}

type ChatSessionListResponse struct {
	Sessions []ChatSessionSummary `json:"sessions"`
}

type ChatSessionDetailResponse struct {
	Session  ChatSessionSummary `json:"session"`
	Messages []ChatMessageItem  `json:"messages"`
}

type ChatTurn struct {
	UserMessage        string
	AssistantContent   string
	AssistantReasoning string
}
