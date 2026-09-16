package ai

type ChatMessage struct {
	Role    string `json:"role"` // "user", "assistant", or "model"
	Content string `json:"content"`
}

type ChatRequest struct {
	SessionID string        `json:"sessionId"`
	Message   string        `json:"message"`
	History   []ChatMessage `json:"history,omitempty"`
}

type ChatResponse struct {
	Response         string   `json:"response"`
	HandoffTriggered bool     `json:"handoffTriggered"`
	ToolsInvoked     []string `json:"toolsInvoked,omitempty"`
	ProviderUsed     string   `json:"providerUsed,omitempty"`
}
