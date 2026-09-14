package service

import (
	"context"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/auth"
	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/repository"
)

type AdminService struct {
	users      *repository.UserRepository
	adminRepo  *repository.AdminRepository
	adminEmail string
}

func NewAdminService(
	users *repository.UserRepository,
	adminRepo *repository.AdminRepository,
	adminEmail string,
) *AdminService {
	return &AdminService{
		users:      users,
		adminRepo:  adminRepo,
		adminEmail: strings.ToLower(strings.TrimSpace(adminEmail)),
	}
}

func (s *AdminService) ListUsers(ctx context.Context) ([]model.AdminUserListItem, error) {
	items, err := s.adminRepo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].IsAdmin = strings.EqualFold(strings.TrimSpace(items[i].Email), s.adminEmail)
	}
	return items, nil
}

func (s *AdminService) GetUserDetail(ctx context.Context, userID int64) (model.AdminUserDetail, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return model.AdminUserDetail{}, err
	}
	tokenSummary, err := s.adminRepo.GetUserTokenSummary(ctx, userID)
	if err != nil {
		return model.AdminUserDetail{}, err
	}
	tokenByDay, err := s.adminRepo.GetUserTokenByDay(ctx, userID)
	if err != nil {
		return model.AdminUserDetail{}, err
	}
	recentChats, err := s.GetUserChats(ctx, userID)
	if err != nil {
		return model.AdminUserDetail{}, err
	}
	return model.AdminUserDetail{
		User:         toUserInfo(user, s.adminEmail),
		TokenSummary: tokenSummary,
		TokenByDay:   tokenByDay,
		RecentChats:  recentChats,
	}, nil
}

func (s *AdminService) GetUserChats(ctx context.Context, userID int64) ([]model.AdminChatSessionItem, error) {
	sessions, err := s.adminRepo.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for idx := range sessions {
		messages, listErr := s.adminRepo.ListSessionMessages(ctx, sessions[idx].ID)
		if listErr != nil {
			return nil, listErr
		}
		sessions[idx].Messages = messages
	}
	return sessions, nil
}

func (s *AdminService) UpdateUserDailyTokenLimit(ctx context.Context, userID int64, dailyTokenLimit int64) error {
	if dailyTokenLimit <= 0 {
		return fmt.Errorf("每日 token 上限必须大于 0")
	}
	if dailyTokenLimit > 2_000_000_000 {
		return fmt.Errorf("每日 token 上限过大，请设置更合理的值")
	}
	return s.users.UpdateDailyTokenLimit(ctx, userID, dailyTokenLimit)
}

func (s *AdminService) UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error {
	trimmed := strings.TrimSpace(newPassword)
	if len(trimmed) < 6 {
		return fmt.Errorf("密码至少需要 6 位")
	}
	hash, err := auth.HashPassword(trimmed)
	if err != nil {
		return fmt.Errorf("密码处理失败")
	}
	return s.users.UpdatePasswordHash(ctx, userID, hash)
}

func (s *AdminService) GetVisitStats(ctx context.Context) (model.AdminVisitStats, error) {
	return s.adminRepo.GetVisitStats(ctx)
}

func (s *AdminService) GetTokenOverview(ctx context.Context) (model.AdminTokenOverview, error) {
	return s.adminRepo.GetTokenOverview(ctx)
}

func (s *AdminService) RecordVisit(ctx context.Context, visitorKey string, userID int64) error {
	return s.adminRepo.RecordVisit(ctx, visitorKey, userID)
}
