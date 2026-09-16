package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         int
	NodeEnv      string
	DatabaseURL  string
	CORSOrigins  []string

	// Auth & Admin
	SessionSecret string
	AdminPIN      string
	AdminPassword string
	OwnerPhone    string

	// Razorpay
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string

	// Meta WhatsApp
	MetaWhatsAppToken string
	MetaPhoneNumberID string
	MetaOwnerPhone    string

	// ResAvenue Channel Manager
	ResAvenueUsername      string
	ResAvenuePassword      string
	ResAvenueIDContext     string
	ResAvenuePushURL       string
	ResAvenuePushAuth      string
	ResAvenueTarget        string

	// AI Concierge
	GeminiAPIKeys []string
	GeminiModels  []string
	GroqAPIKeys   []string
}

func Load() *Config {
	port, err := strconv.Atoi(getEnv("PORT", "3001"))
	if err != nil || port <= 0 {
		port = 3001
	}

	nodeEnv := getEnv("NODE_ENV", "development")

	// Collect CORS origins
	var corsOrigins []string
	rawCORS := getEnv("CORS_ORIGIN", "")
	if rawCORS != "" {
		for _, o := range strings.Split(rawCORS, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	} else {
		corsOrigins = []string{
			"https://quadishotels.com",
			"https://www.quadishotels.com",
			"http://localhost:5173",
			"http://localhost:3000",
		}
	}

	// Collect Gemini API keys
	var geminiKeys []string
	if raw := getEnv("GEMINI_API_KEYS", ""); raw != "" {
		for _, k := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(k); trimmed != "" {
				geminiKeys = append(geminiKeys, trimmed)
			}
		}
	}
	if single := getEnv("GEMINI_API_KEY", ""); single != "" {
		geminiKeys = append(geminiKeys, single)
	}
	for i := 1; i <= 10; i++ {
		if k := os.Getenv("GEMINI_API_KEY_" + strconv.Itoa(i)); k != "" {
			geminiKeys = append(geminiKeys, strings.TrimSpace(k))
		}
	}

	// Collect Gemini models
	rawModels := getEnv("GEMINI_MODELS", "gemini-2.5-flash,gemini-2.5-flash-lite,gemini-2.0-flash")
	var geminiModels []string
	for _, m := range strings.Split(rawModels, ",") {
		if trimmed := strings.TrimSpace(m); trimmed != "" {
			geminiModels = append(geminiModels, trimmed)
		}
	}

	// Collect Groq API keys
	var groqKeys []string
	if raw := getEnv("GROQ_API_KEYS", ""); raw != "" {
		for _, k := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(k); trimmed != "" {
				groqKeys = append(groqKeys, trimmed)
			}
		}
	}
	if single := getEnv("GROQ_API_KEY", ""); single != "" {
		groqKeys = append(groqKeys, single)
	}
	for i := 1; i <= 10; i++ {
		if k := os.Getenv("GROQ_API_KEY_" + strconv.Itoa(i)); k != "" {
			groqKeys = append(groqKeys, strings.TrimSpace(k))
		}
	}

	return &Config{
		Port:                  port,
		NodeEnv:               nodeEnv,
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		CORSOrigins:           corsOrigins,
		SessionSecret:         getEnv("SESSION_SECRET", ""),
		AdminPIN:              getEnv("ADMIN_PIN", "998877"),
		AdminPassword:         getEnv("ADMIN_PASSWORD", ""),
		OwnerPhone:            getEnv("OWNER_WHATSAPP_PHONE", "919217373532"),
		RazorpayKeyID:         getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:     getEnv("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", ""),
		MetaWhatsAppToken:     getEnv("META_WHATSAPP_TOKEN", ""),
		MetaPhoneNumberID:     getEnv("META_PHONE_NUMBER_ID", ""),
		MetaOwnerPhone:        getEnv("META_OWNER_PHONE", "919217373532"),
		ResAvenueUsername:      getEnv("RESAVENUE_OTA_USERNAME", ""),
		ResAvenuePassword:      getEnv("RESAVENUE_OTA_PASSWORD", ""),
		ResAvenueIDContext:     getEnv("RESAVENUE_OTA_ID_CONTEXT", ""),
		ResAvenuePushURL:       getEnv("RESAVENUE_PUSH_URL", ""),
		ResAvenuePushAuth:      getEnv("RESAVENUE_PUSH_AUTHORIZATION", ""),
		ResAvenueTarget:        getEnv("RESAVENUE_TARGET", "Production"),
		GeminiAPIKeys:         geminiKeys,
		GeminiModels:          geminiModels,
		GroqAPIKeys:           groqKeys,
	}
}

func (c *Config) IsProduction() bool {
	return c.NodeEnv == "production"
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
