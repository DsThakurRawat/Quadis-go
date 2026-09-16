package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"quadis-backend-go/internal/config"
)

const DevSessionSecret = "quadis-dev-only-session-secret"

type SessionPayload struct {
	Sub   string  `json:"sub"`
	Email string  `json:"email"`
	Exp   int64   `json:"exp"`
	Role  *string `json:"role,omitempty"`
}

type SessionService struct {
	cfg *config.Config
}

func NewSessionService(cfg *config.Config) *SessionService {
	return &SessionService{cfg: cfg}
}

func (s *SessionService) getSecret() string {
	if s.cfg.SessionSecret != "" {
		return s.cfg.SessionSecret
	}
	if s.cfg.IsProduction() {
		return ""
	}
	return DevSessionSecret
}

// base64url encodes bytes to base64url without padding
func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeBase64url decodes unpadded base64url
func decodeBase64url(str string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(str)
}

// SignSession creates a compact HMAC-signed session token (payload.signature)
func (s *SessionService) SignSession(sub, email string, role *string, ttlSeconds int64) (string, error) {
	secret := s.getSecret()
	if secret == "" {
		return "", nil
	}

	if ttlSeconds <= 0 {
		ttlSeconds = 7 * 24 * 60 * 60 // 7 days default
	}

	body := SessionPayload{
		Sub:   sub,
		Email: email,
		Exp:   time.Now().Unix() + ttlSeconds,
		Role:  role,
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	dataStr := base64url(jsonBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(dataStr))
	sigStr := base64url(mac.Sum(nil))

	return dataStr + "." + sigStr, nil
}

// VerifySession verifies and extracts the session payload from a token
func (s *SessionService) VerifySession(token string) *SessionPayload {
	secret := s.getSecret()
	if secret == "" {
		return nil
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil
	}

	dataStr, sigStr := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(dataStr))
	expectedSig := base64url(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(sigStr), []byte(expectedSig)) != 1 {
		return nil
	}

	payloadBytes, err := decodeBase64url(dataStr)
	if err != nil {
		return nil
	}

	var payload SessionPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil
	}

	if payload.Exp < time.Now().Unix() {
		return nil // Expired
	}

	return &payload
}
