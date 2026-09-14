package service

import (
	"context"
	"fmt"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/repository"
)

type UsageService struct {
	usageRepo *repository.UsageRepository
	userRepo  *repository.UserRepository
}

func NewUsageService(usageRepo *repository.UsageRepository, userRepo *repository.UserRepository) *UsageService {
	return &UsageService{
		usageRepo: usageRepo,
		userRepo:  userRepo,
	}
}

func (s *UsageService) RecordChatUsage(ctx context.Context, userID int64, usage *model.TokenUsage, llmModel string) error {
	if usage == nil || userID <= 0 {
		return nil
	}
	return s.usageRepo.Insert(ctx, userID, *usage, llmModel)
}

func (s *UsageService) GetSummary(ctx context.Context, userID int64, days int) (model.UsageSummary, error) {
	return s.usageRepo.GetSummary(ctx, userID, days)
}

func (s *UsageService) EnsureWithinDailyLimit(ctx context.Context, userID int64) error {
	if userID <= 0 || s.userRepo == nil || s.usageRepo == nil {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.DailyTokenLimit <= 0 {
		return nil
	}
	todayTotal, err := s.usageRepo.GetUserTodayTotal(ctx, userID)
	if err != nil {
		return err
	}
	if todayTotal >= user.DailyTokenLimit {
		return fmt.Errorf("今日 token 已达到上限（%d）", user.DailyTokenLimit)
	}
	return nil
}
