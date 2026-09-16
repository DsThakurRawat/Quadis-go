package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
)

type AIService struct {
	repo           repository.Repository
	cfg            *config.Config
	fallback       *FallbackEngine
	httpClient     *http.Client
	geminiKeyIndex int
	groqKeyIndex   int
	cooldowns      map[string]time.Time
	mu             sync.Mutex
}

func NewAIService(repo repository.Repository, cfg *config.Config) *AIService {
	return &AIService{
		repo:       repo,
		cfg:        cfg,
		fallback:   NewFallbackEngine(repo),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cooldowns:  make(map[string]time.Time),
	}
}

func (s *AIService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var resp *ChatResponse
	var err error

	// 1. Try Gemini models
	if len(s.cfg.GeminiAPIKeys) > 0 && len(s.cfg.GeminiModels) > 0 {
		for _, model := range s.cfg.GeminiModels {
			if s.isCoolingDown("gemini:" + model) {
				continue
			}
			resp, err = s.callGemini(ctx, model, req)
			if err == nil && resp != nil {
				break
			}
			s.setCooldown("gemini:"+model, 5*time.Minute)
		}
	}

	// 2. Try Groq fallback
	if resp == nil && len(s.cfg.GroqAPIKeys) > 0 {
		if !s.isCoolingDown("groq") {
			resp, err = s.callGroq(ctx, "llama-3.3-70b-versatile", req)
			if err != nil {
				s.setCooldown("groq", 5*time.Minute)
			}
		}
	}

	// 3. Deterministic fallback
	if resp == nil {
		fb := s.fallback.GenerateFallbackResponse(ctx, req.Message)
		resp = &fb
	}

	// Log chat interaction
	if req.SessionID != "" {
		_ = s.repo.CreateChatLog(ctx, &domain.ChatLogRecord{
			SessionID:        req.SessionID,
			UserMessage:      req.Message,
			BotResponse:      resp.Response,
			ToolsInvoked:     resp.ToolsInvoked,
			HandoffTriggered: resp.HandoffTriggered,
		})
	}

	return resp, nil
}

func (s *AIService) isCoolingDown(target string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.cooldowns[target]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(s.cooldowns, target)
		return false
	}
	return true
}

func (s *AIService) setCooldown(target string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cooldowns[target] = time.Now().Add(duration)
}

func (s *AIService) getNextGeminiKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cfg.GeminiAPIKeys) == 0 {
		return ""
	}
	key := s.cfg.GeminiAPIKeys[s.geminiKeyIndex%len(s.cfg.GeminiAPIKeys)]
	s.geminiKeyIndex++
	return key
}

func (s *AIService) getNextGroqKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cfg.GroqAPIKeys) == 0 {
		return ""
	}
	key := s.cfg.GroqAPIKeys[s.groqKeyIndex%len(s.cfg.GroqAPIKeys)]
	s.groqKeyIndex++
	return key
}

func (s *AIService) callGemini(ctx context.Context, model string, req ChatRequest) (*ChatResponse, error) {
	key := s.getNextGeminiKey()
	if key == "" {
		return nil, fmt.Errorf("no gemini key")
	}

	systemInstruction := "You are the AI Concierge for Quadis Hotels, premium boutique hotels located across New Delhi and Noida. Answer concisely, warmly, and helpfully."

	type contentPart struct {
		Text string `json:"text"`
	}
	type contentMessage struct {
		Role  string        `json:"role"`
		Parts []contentPart `json:"parts"`
	}

	var contents []contentMessage
	for _, h := range req.History {
		role := h.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, contentMessage{
			Role:  role,
			Parts: []contentPart{{Text: h.Content}},
		})
	}
	contents = append(contents, contentMessage{
		Role:  "user",
		Parts: []contentPart{{Text: req.Message}},
	})

	geminiReqBody := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []contentPart{{Text: systemInstruction}},
		},
		"contents": contents,
	}

	bodyBytes, err := json.Marshal(geminiReqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, key)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("gemini status %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty gemini candidates")
	}

	reply := geminiResp.Candidates[0].Content.Parts[0].Text
	return &ChatResponse{
		Response:     reply,
		ProviderUsed: "gemini:" + model,
	}, nil
}

func (s *AIService) callGroq(ctx context.Context, model string, req ChatRequest) (*ChatResponse, error) {
	key := s.getNextGroqKey()
	if key == "" {
		return nil, fmt.Errorf("no groq key")
	}

	type groqMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	var messages []groqMessage
	messages = append(messages, groqMessage{
		Role:    "system",
		Content: "You are the AI Concierge for Quadis Hotels, premium boutique hotels in New Delhi and Noida. Answer concisely and warmly.",
	})
	for _, h := range req.History {
		messages = append(messages, groqMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}
	messages = append(messages, groqMessage{
		Role:    "user",
		Content: req.Message,
	})

	groqReqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	bodyBytes, err := json.Marshal(groqReqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("groq status %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&groqResp); err != nil {
		return nil, err
	}

	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("empty groq choices")
	}

	return &ChatResponse{
		Response:     groqResp.Choices[0].Message.Content,
		ProviderUsed: "groq:" + model,
	}, nil
}
