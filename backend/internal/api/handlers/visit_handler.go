package handlers

import (
	"crypto/sha1"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"strings"

	"gemini-clone/backend/internal/middleware"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type VisitHandler struct {
	adminService *service.AdminService
	jwtSecret    string
}

func NewVisitHandler(adminService *service.AdminService, jwtSecret string) *VisitHandler {
	return &VisitHandler{
		adminService: adminService,
		jwtSecret:    jwtSecret,
	}
}

func (h *VisitHandler) PostVisit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	userID := middleware.ParseAuthUserID(r, h.jwtSecret)
	visitorKey := buildVisitorKey(r, userID)
	if err := h.adminService.RecordVisit(r.Context(), visitorKey, userID); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func buildVisitorKey(r *http.Request, userID int64) string {
	if userID > 0 {
		return "user:" + strconv.FormatInt(userID, 10)
	}
	clientIP := extractClientIP(r)
	ua := strings.TrimSpace(r.UserAgent())
	hashSource := clientIP + "|" + ua
	sum := sha1.Sum([]byte(hashSource))
	return "anon:" + hex.EncodeToString(sum[:])
}

func extractClientIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
