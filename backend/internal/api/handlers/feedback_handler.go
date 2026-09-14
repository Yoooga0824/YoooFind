package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gemini-clone/backend/internal/middleware"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type FeedbackHandler struct {
	feedbackService *service.FeedbackService
	jwtSecret       string
}

func NewFeedbackHandler(feedbackService *service.FeedbackService, jwtSecret string) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
		jwtSecret:       jwtSecret,
	}
}

func (h *FeedbackHandler) PostFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}

	var req model.FeedbackSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid request body"}})
		return
	}

	userID := middleware.ParseAuthUserID(r, h.jwtSecret)
	id, err := h.feedbackService.SubmitFeedback(r.Context(), userID, req.Title, req.Content)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "ok"})
}

func (h *FeedbackHandler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	items, err := h.feedbackService.ListFeedback(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, model.FeedbackListResponse{Feedback: items})
}

func (h *FeedbackHandler) HandleFeedbackActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/feedback/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, model.ErrorEnvelope{Error: model.ErrorBody{Message: "feedback id is required"}})
		return
	}

	feedbackID, err := strconv.ParseInt(strings.TrimSpace(path), 10, 64)
	if err != nil || feedbackID <= 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid feedback id"}})
		return
	}

	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}

	var req model.FeedbackStatusPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid request body"}})
		return
	}
	if strings.TrimSpace(req.Status) != "read" {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "unsupported status"}})
		return
	}

	if err := h.feedbackService.MarkFeedbackRead(r.Context(), feedbackID); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
