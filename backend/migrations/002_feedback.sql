CREATE TABLE IF NOT EXISTS feedback (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NULL,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  user_email VARCHAR(255) NOT NULL DEFAULT '',
  user_display_name VARCHAR(100) NOT NULL DEFAULT '',
  status ENUM('new', 'read') NOT NULL DEFAULT 'new',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_feedback_status_created (status, created_at),
  INDEX idx_feedback_user_id (user_id)
);
