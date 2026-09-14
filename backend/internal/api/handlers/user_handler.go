package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gemini-clone/backend/internal/middleware"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/service"
)

type UserHandler struct {
	userService     *service.UserService
	avatarUploadDir string
	avatarMaxBytes  int64
}

func NewUserHandler(userService *service.UserService, avatarUploadDir string, avatarMaxBytes int64) *UserHandler {
	return &UserHandler{
		userService:     userService,
		avatarUploadDir: avatarUploadDir,
		avatarMaxBytes:  avatarMaxBytes,
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	info, err := h.userService.GetMe(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *UserHandler) PatchMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "invalid JSON body"}})
		return
	}
	info, err := h.userService.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *UserHandler) PostAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorEnvelope{Error: model.ErrorBody{Message: "method not allowed"}})
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if err := r.ParseMultipartForm(h.avatarMaxBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "头像上传失败：文件过大或格式错误"}})
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "请上传头像文件"}})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: "头像仅支持图片格式"}})
		return
	}

	if err := os.MkdirAll(h.avatarUploadDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorEnvelope{Error: model.ErrorBody{Message: "创建上传目录失败"}})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	fileName := fmt.Sprintf("user-%d-%d%s", userID, time.Now().UnixNano(), ext)
	targetPath := filepath.Join(h.avatarUploadDir, fileName)

	out, err := os.Create(targetPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorEnvelope{Error: model.ErrorBody{Message: "保存头像失败"}})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(file, h.avatarMaxBytes)); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorEnvelope{Error: model.ErrorBody{Message: "写入头像失败"}})
		return
	}

	avatarURL := "/uploads/avatars/" + fileName
	info, err := h.userService.UpdateAvatarPath(r.Context(), userID, avatarURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorEnvelope{Error: model.ErrorBody{Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, info)
}
