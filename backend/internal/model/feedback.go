package model

type FeedbackSubmitRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type FeedbackItem struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id,omitempty"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	UserEmail       string `json:"user_email"`
	UserDisplayName string `json:"user_display_name"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

type FeedbackListResponse struct {
	Feedback []FeedbackItem `json:"feedback"`
}

type FeedbackStatusPatchRequest struct {
	Status string `json:"status"`
}
