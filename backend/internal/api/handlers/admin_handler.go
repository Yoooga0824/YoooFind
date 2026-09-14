package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	users, err := h.adminService.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (h *AdminHandler) GetVisitStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	stats, err := h.adminService.GetVisitStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) GetTokenOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	overview, err := h.adminService.GetTokenOverview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (h *AdminHandler) HandleUserActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, model.ErrorEnvelope{Error: model.ErrorBody{Message: "user id is required"}})
		return
	}
	segments := strings.Split(path, "/")
	userID, err := strconv.ParseInt(strings.TrimSpace(segments[0]), 10, 64)
	if err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid user id"}})
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		h.getUserDetail(w, r, userID)
	case len(segments) == 2 && segments[1] == "token-limit" && r.Method == http.MethodPatch:
		h.patchUserTokenLimit(w, r, userID)
	case len(segments) == 2 && segments[1] == "password" && r.Method == http.MethodPatch:
		h.patchUserPassword(w, r, userID)
	case len(segments) == 2 && segments[1] == "chats" && r.Method == http.MethodGet:
		h.getUserChats(w, r, userID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
	}
}

func (h *AdminHandler) getUserDetail(w http.ResponseWriter, r *http.Request, userID int64) {
	detail, err := h.adminService.GetUserDetail(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *AdminHandler) getUserChats(w http.ResponseWriter, r *http.Request, userID int64) {
	chats, err := h.adminService.GetUserChats(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": chats})
}

func (h *AdminHandler) patchUserTokenLimit(w http.ResponseWriter, r *http.Request, userID int64) {
	var req struct {
		DailyTokenLimit int64 `json:"daily_token_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid JSON body"}})
		return
	}
	if err := h.adminService.UpdateUserDailyTokenLimit(r.Context(), userID, req.DailyTokenLimit); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminHandler) patchUserPassword(w http.ResponseWriter, r *http.Request, userID int64) {
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid JSON body"}})
		return
	}
	if err := h.adminService.UpdateUserPassword(r.Context(), userID, req.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
