package model

type User struct {
	ID              int64
	Email           string
	PasswordHash    string
	DisplayName     string
	FullName        string
	Bio             string
	AvatarPath      string
	DailyTokenLimit int64
	CreatedAt       string
	UpdatedAt       string
}

type UserInfo struct {
	ID              int64  `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"display_name"`
	FullName        string `json:"full_name"`
	Bio             string `json:"bio"`
	AvatarURL       string `json:"avatar_url"`
	DailyTokenLimit int64  `json:"daily_token_limit"`
	IsAdmin         bool   `json:"is_admin"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	Bio         string `json:"bio"`
}
