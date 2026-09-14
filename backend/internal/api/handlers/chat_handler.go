package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"gemini-clone/backend/internal/middleware"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

// ChatHandler handles /api/chat endpoint.
type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// PostChat accepts frontend message and returns OpenAI-compatible JSON.
func (h *ChatHandler) PostChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "method not allowed"},
		})
		return
	}

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "invalid JSON body"},
		})
		return
	}

	if wantsStreamResponse(r) {
		h.postChatStream(w, r, req)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	replies, session, err := h.chatService.ReplyMulti(
		r.Context(),
		userID,
		req.SessionID,
		req.Message,
		req.Models,
		model.ReplyOptions{DeepSearch: req.DeepSearch},
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: err.Error()},
		})
		return
	}

	choices := make([]model.Choice, 0, len(replies))
	for _, reply := range replies {
		choices = append(choices, model.Choice{
			Message: model.Message{
				Role:             "assistant",
				Content:          reply.Content,
				ReasoningContent: reply.ReasoningContent,
				Model:            reply.Model,
			},
		})
	}
	primaryUsage := replies[0].Usage
	resp := model.OpenAICompatibleResponse{
		Choices: choices,
		Usage:   primaryUsage,
		Session: &session,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) postChatStream(w http.ResponseWriter, r *http.Request, req model.ChatRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "streaming not supported by server"},
		})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var sendMu sync.Mutex
	sendEvent := func(payload any) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	userID := middleware.UserIDFromContext(r.Context())
	replies, session, err := h.chatService.StreamReplyMulti(
		r.Context(),
		userID,
		req.SessionID,
		req.Message,
		req.Models,
		model.ReplyOptions{DeepSearch: req.DeepSearch},
		func(session model.ChatSessionSummary) error {
			return sendEvent(map[string]any{
				"type":    "session",
				"session": session,
			})
		},
		func(modelKey string, delta model.AssistantReplyDelta) error {
			return sendEvent(map[string]string{
				"type":              "delta",
				"model":             modelKey,
				"content":           delta.Content,
				"reasoning_content": delta.ReasoningContent,
			})
		},
		func(modelKey string, runErr error) error {
			return sendEvent(map[string]string{
				"type":  "model_error",
				"model": modelKey,
				"error": runErr.Error(),
			})
		},
	)
	if err != nil {
		_ = sendEvent(map[string]string{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	selectedModel := ""
	selectedContent := ""
	selectedReasoning := ""
	var selectedUsage *model.TokenUsage
	if len(replies) > 0 {
		selectedModel = replies[0].Model
		selectedContent = replies[0].Content
		selectedReasoning = replies[0].ReasoningContent
		selectedUsage = replies[0].Usage
	}
	_ = sendEvent(map[string]any{
		"type":              "done",
		"content":           selectedContent,
		"reasoning_content": selectedReasoning,
		"usage":             selectedUsage,
		"model_responses":   replies,
		"selected_model":    selectedModel,
		"session":           session,
	})
}

func (h *ChatHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "method not allowed"},
		})
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	sessions, err := h.chatService.ListSessions(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, model.ChatSessionListResponse{Sessions: sessions})
}

func (h *ChatHandler) GetSessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "method not allowed"},
		})
		return
	}
	rawID := strings.TrimPrefix(r.URL.Path, "/api/chat/sessions/")
	sessionID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || sessionID <= 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: "invalid session id"},
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if r.Method == http.MethodDelete {
		if err := h.chatService.DeleteSession(r.Context(), userID, sessionID); err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, model.ErrorEnvelope{
				Error: model.ErrorBody{Message: err.Error()},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	session, messages, err := h.chatService.GetSessionDetail(r.Context(), userID, sessionID)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, model.ErrorEnvelope{
			Error: model.ErrorBody{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, model.ChatSessionDetailResponse{
		Session:  session,
		Messages: messages,
	})
}

func wantsStreamResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/event-stream")
}

func firstModelOrFallback(models []string) string {
	for _, item := range models {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
