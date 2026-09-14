package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/repository"
)

type FeedbackService struct {
	feedbackRepo *repository.FeedbackRepository
	userRepo     *repository.UserRepository
}

func NewFeedbackService(
	feedbackRepo *repository.FeedbackRepository,
	userRepo *repository.UserRepository,
) *FeedbackService {
	return &FeedbackService{
		feedbackRepo: feedbackRepo,
		userRepo:     userRepo,
	}
}

func (s *FeedbackService) SubmitFeedback(
	ctx context.Context,
	userID int64,
	title, content string,
) (int64, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" {
		return 0, fmt.Errorf("标题不能为空")
	}
	if len([]rune(title)) > 200 {
		return 0, fmt.Errorf("标题不能超过 200 字")
	}
	if content == "" {
		return 0, fmt.Errorf("内容不能为空")
	}
	if len([]rune(content)) > 5000 {
		return 0, fmt.Errorf("内容不能超过 5000 字")
	}

	userEmail := ""
	userDisplayName := ""
	if userID > 0 {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err == nil {
			userEmail = user.Email
			userDisplayName = user.DisplayName
		}
	}

	return s.feedbackRepo.Create(ctx, userID, title, content, userEmail, userDisplayName)
}

func (s *FeedbackService) ListFeedback(ctx context.Context) ([]model.FeedbackItem, error) {
	return s.feedbackRepo.ListAll(ctx)
}

func (s *FeedbackService) MarkFeedbackRead(ctx context.Context, feedbackID int64) error {
	if feedbackID <= 0 {
		return fmt.Errorf("invalid feedback id")
	}
	err := s.feedbackRepo.UpdateStatus(ctx, feedbackID, "read")
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("反馈不存在")
	}
	return err
}
