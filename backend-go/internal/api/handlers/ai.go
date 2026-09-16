package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/ai"
)

type AIHandler struct {
	aiService *ai.AIService
	repo      repository.Repository
}

func NewAIHandler(aiService *ai.AIService, repo repository.Repository) *AIHandler {
	return &AIHandler{aiService: aiService, repo: repo}
}

type chatTurnPayload struct {
	SessionID string           `json:"sessionId,omitempty"`
	Message   string           `json:"message"`
	History   []ai.ChatMessage `json:"history,omitempty"`
}

// Chat handles POST /api/ai/chat
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var payload chatTurnPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(payload.Message) == "" {
		ErrorJSON(w, http.StatusBadRequest, "Message text is required")
		return
	}

	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", time.Now().UnixMilli())
	}

	resp, err := h.aiService.Chat(r.Context(), ai.ChatRequest{
		SessionID: sessionID,
		Message:   payload.Message,
		History:   payload.History,
	})

	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "AI processing failure")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"reply":            resp.Response,
			"toolsInvoked":     resp.ToolsInvoked,
			"handoffTriggered": resp.HandoffTriggered,
			"sessionId":        sessionID,
		},
	})
}

// Logs handles GET /api/ai/logs (requireAdmin)
func (h *AIHandler) Logs(w http.ResponseWriter, r *http.Request) {
	// In the memory store / PG store, logs can be returned
	// Return empty list or stored logs
	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   0,
		"data":    []interface{}{},
	})
}
