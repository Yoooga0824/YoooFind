package model

type AdminUserListItem struct {
	ID              int64  `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"display_name"`
	FullName        string `json:"full_name"`
	Bio             string `json:"bio"`
	AvatarURL       string `json:"avatar_url"`
	DailyTokenLimit int64  `json:"daily_token_limit"`
	CreatedAt       string `json:"created_at"`
	LastActiveAt    string `json:"last_active_at"`
	TodayTokens     int64  `json:"today_tokens"`
	TotalTokens     int64  `json:"total_tokens"`
	SessionCount    int64  `json:"session_count"`
	MessageCount    int64  `json:"message_count"`
	IsAdmin         bool   `json:"is_admin"`
}

type AdminUserTokenSummary struct {
	TodayTokens int64 `json:"today_tokens"`
	TotalTokens int64 `json:"total_tokens"`
}

type AdminUserDetail struct {
	User         UserInfo               `json:"user"`
	TokenSummary AdminUserTokenSummary  `json:"token_summary"`
	TokenByDay   []UsagePoint           `json:"token_by_day"`
	RecentChats  []AdminChatSessionItem `json:"recent_chats"`
}

type AdminChatSessionItem struct {
	ID        int64                 `json:"id"`
	Title     string                `json:"title"`
	UpdatedAt string                `json:"updated_at"`
	Messages  []AdminChatMessageRow `json:"messages"`
}

type AdminChatMessageRow struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
	Model            string `json:"model,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type AdminVisitPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type AdminVisitStats struct {
	TotalUniqueVisitors int64             `json:"total_unique_visitors"`
	TodayUniqueVisitors int64             `json:"today_unique_visitors"`
	LoggedInVisitors    int64             `json:"logged_in_visitors"`
	AnonymousVisitors   int64             `json:"anonymous_visitors"`
	DailyTrend          []AdminVisitPoint `json:"daily_trend"`
}

type AdminTokenOverview struct {
	TodayTotalTokens int64                       `json:"today_total_tokens"`
	HistoryTotal     int64                       `json:"history_total_tokens"`
	DailyTotal       []UsagePoint                `json:"daily_total"`
	Users            []AdminUserTokenSummaryItem `json:"users"`
}

type AdminUserTokenSummaryItem struct {
	UserID      int64  `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	TodayTokens int64  `json:"today_tokens"`
	TotalTokens int64  `json:"total_tokens"`
}
