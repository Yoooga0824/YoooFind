package handlers

import (
	"net/http"
	"strconv"

	"gemini-clone/backend/internal/middleware"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type UsageHandler struct {
	usageService *service.UsageService
}

func NewUsageHandler(usageService *service.UsageService) *UsageHandler {
	return &UsageHandler{usageService: usageService}
}

func (h *UsageHandler) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	days := 30
	if rawDays := r.URL.Query().Get("days"); rawDays != "" {
		if parsed, err := strconv.Atoi(rawDays); err == nil {
			days = parsed
		}
	}
	summary, err := h.usageService.GetSummary(r.Context(), userID, days)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
