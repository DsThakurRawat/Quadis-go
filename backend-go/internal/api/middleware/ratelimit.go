package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type clientVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*clientVisitor
	rate     rate.Limit
	burst    int
	isProd   bool
	resBody  []byte
}

// NewIPRateLimiter creates a rate limiter allowing `maxReqs` per `window`.
// If responseBody is nil, defaults to standard JSON error.
func NewIPRateLimiter(maxReqs int, window time.Duration, isProd bool, responseBody []byte) *IPRateLimiter {
	r := rate.Every(window / time.Duration(maxReqs))
	limiter := &IPRateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     r,
		burst:    maxReqs,
		isProd:   isProd,
		resBody:  responseBody,
	}

	// Periodic cleanup of stale visitors
	go func() {
		ticker := time.NewTicker(time.Minute * 5)
		for range ticker.C {
			limiter.mu.Lock()
			for ip, v := range limiter.visitors {
				if time.Since(v.lastSeen) > window*2 {
					delete(limiter.visitors, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (l *IPRateLimiter) getClientIP(r *http.Request) string {
	if l.isProd {
		// Trust 1 hop: check X-Forwarded-For or X-Real-IP
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *IPRateLimiter) getVisitor(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, exists := l.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(l.rate, l.burst)
		l.visitors[ip] = &clientVisitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (l *IPRateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := l.getClientIP(r)
		limiter := l.getVisitor(ip)
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			if len(l.resBody) > 0 {
				_, _ = w.Write(l.resBody)
			} else {
				_, _ = w.Write([]byte(`{"success":false,"error":"Too many requests. Try again later."}`))
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
