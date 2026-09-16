package middleware

import (
	"net/http"
	"regexp"

	"github.com/go-chi/cors"
)

var (
	devLocalhost = regexp.MustCompile(`^http://localhost:\d+$`)
	dev127       = regexp.MustCompile(`^http://127\.0\.0\.1:\d+$`)
)

// NewCORS creates a Chi CORS middleware configured with strict allowed origins.
func NewCORS(allowedOrigins []string, isProd bool) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o != "" {
			allowedMap[o] = true
		}
	}

	return cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if origin == "" {
				return true
			}
			if allowedMap[origin] {
				return true
			}
			if !isProd {
				if devLocalhost.MatchString(origin) || dev127.MatchString(origin) {
					return true
				}
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Razorpay-Signature", "X-Simulated-Webhook"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
