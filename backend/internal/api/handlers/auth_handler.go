package handlers

import (
	"encoding/json"
	"net/http"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) PostRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	var req model.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid JSON body"}})
		return
	}
	resp, err := h.authService.Register(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) PostLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	var req model.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid JSON body"}})
		return
	}
	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
