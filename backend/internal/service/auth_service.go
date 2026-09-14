package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/auth"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/repository"
)

type AuthService struct {
	users          *repository.UserRepository
	jwtSecret      string
	jwtExpiryHours int
	adminEmail     string
}

func NewAuthService(
	users *repository.UserRepository,
	jwtSecret string,
	jwtExpiryHours int,
	adminEmail string,
) *AuthService {
	return &AuthService{
		users:          users,
		jwtSecret:      jwtSecret,
		jwtExpiryHours: jwtExpiryHours,
		adminEmail:     strings.ToLower(strings.TrimSpace(adminEmail)),
	}
}

func (s *AuthService) Register(ctx context.Context, req model.AuthRequest) (model.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return model.AuthResponse{}, fmt.Errorf("邮箱和密码不能为空")
	}
	if len(password) < 6 {
		return model.AuthResponse{}, fmt.Errorf("密码至少需要 6 位")
	}
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return model.AuthResponse{}, fmt.Errorf("该邮箱已被注册")
	} else if err != sql.ErrNoRows {
		return model.AuthResponse{}, err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("密码处理失败")
	}
	userID, err := s.users.Create(ctx, email, hash)
	if err != nil {
		return model.AuthResponse{}, err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return model.AuthResponse{}, err
	}

	token, err := auth.SignToken(s.jwtSecret, userID, s.jwtExpiryHours)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("生成登录态失败")
	}
	return model.AuthResponse{
		Token: token,
		User:  toUserInfo(user, s.adminEmail),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req model.AuthRequest) (model.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return model.AuthResponse{}, fmt.Errorf("邮箱和密码不能为空")
	}
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.AuthResponse{}, fmt.Errorf("账号或密码错误")
		}
		return model.AuthResponse{}, err
	}
	if !auth.VerifyPassword(password, user.PasswordHash) {
		return model.AuthResponse{}, fmt.Errorf("账号或密码错误")
	}
	token, err := auth.SignToken(s.jwtSecret, user.ID, s.jwtExpiryHours)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("生成登录态失败")
	}
	return model.AuthResponse{
		Token: token,
		User:  toUserInfo(user, s.adminEmail),
	}, nil
}

func toUserInfo(user model.User, adminEmail string) model.UserInfo {
	avatarURL := ""
	if strings.TrimSpace(user.AvatarPath) != "" {
		avatarURL = user.AvatarPath
	}
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = "用户"
	}
	return model.UserInfo{
		ID:              user.ID,
		Email:           user.Email,
		DisplayName:     displayName,
		FullName:        user.FullName,
		Bio:             user.Bio,
		AvatarURL:       avatarURL,
		DailyTokenLimit: user.DailyTokenLimit,
		IsAdmin:         strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(adminEmail)),
	}
}
