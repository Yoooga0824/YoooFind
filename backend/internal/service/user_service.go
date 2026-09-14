package service

import (
	"context"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/repository"
)

type UserService struct {
	users      *repository.UserRepository
	adminEmail string
}

func NewUserService(users *repository.UserRepository, adminEmail string) *UserService {
	return &UserService{
		users:      users,
		adminEmail: strings.ToLower(strings.TrimSpace(adminEmail)),
	}
}

func (s *UserService) GetMe(ctx context.Context, userID int64) (model.UserInfo, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return model.UserInfo{}, err
	}
	return toUserInfo(user, s.adminEmail), nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req model.UpdateProfileRequest) (model.UserInfo, error) {
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Bio = strings.TrimSpace(req.Bio)
	if req.DisplayName == "" {
		req.DisplayName = "用户"
	}
	if len(req.DisplayName) > 32 {
		return model.UserInfo{}, fmt.Errorf("昵称最长 32 个字符")
	}
	if len(req.FullName) > 64 {
		return model.UserInfo{}, fmt.Errorf("姓名最长 64 个字符")
	}
	if len(req.Bio) > 300 {
		return model.UserInfo{}, fmt.Errorf("个人简述最长 300 个字符")
	}
	if err := s.users.UpdateProfile(ctx, userID, req); err != nil {
		return model.UserInfo{}, err
	}
	return s.GetMe(ctx, userID)
}

func (s *UserService) UpdateAvatarPath(ctx context.Context, userID int64, avatarPath string) (model.UserInfo, error) {
	if err := s.users.UpdateAvatarPath(ctx, userID, avatarPath); err != nil {
		return model.UserInfo{}, err
	}
	return s.GetMe(ctx, userID)
}
