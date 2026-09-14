// Frontend runtime config (no secrets here).
const config = {
  BACKEND_BASE_URL: "http://localhost:8080",
  BACKEND_API_URL: "http://localhost:8080/api/chat",
  AUTH_LOGIN_URL: "http://localhost:8080/api/auth/login",
  AUTH_REGISTER_URL: "http://localhost:8080/api/auth/register",
  ME_URL: "http://localhost:8080/api/me",
  AVATAR_UPLOAD_URL: "http://localhost:8080/api/me/avatar",
  USAGE_URL: "http://localhost:8080/api/usage",
  CHAT_SESSIONS_URL: "http://localhost:8080/api/chat/sessions",
  VISIT_URL: "http://localhost:8080/api/visit",
  ADMIN_USERS_URL: "http://localhost:8080/api/admin/users",
  ADMIN_VISIT_STATS_URL: "http://localhost:8080/api/admin/stats/visits",
  ADMIN_TOKEN_STATS_URL: "http://localhost:8080/api/admin/stats/tokens",
  FEEDBACK_URL: "http://localhost:8080/api/feedback",
  ADMIN_FEEDBACK_URL: "http://localhost:8080/api/admin/feedback",
};

export default config;
