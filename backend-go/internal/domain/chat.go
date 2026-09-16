package domain

import "time"

type ChatLogRecord struct {
	ID               string    `json:"id" db:"id"`
	SessionID        string    `json:"session_id" db:"session_id"`
	UserMessage      string    `json:"user_message" db:"user_message"`
	BotResponse      string    `json:"bot_response" db:"bot_response"`
	ToolsInvoked     []string  `json:"tools_invoked,omitempty" db:"tools_invoked"`
	HandoffTriggered bool      `json:"handoff_triggered" db:"handoff_triggered"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
