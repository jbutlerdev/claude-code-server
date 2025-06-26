package api

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jbutlerdev/claude-code-server/internal/config"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Implement http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

func (api *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/health") ||
			strings.HasPrefix(r.URL.Path, "/api/ready") ||
			strings.HasPrefix(r.URL.Path, "/api/openapi.json") ||
			strings.HasSuffix(r.URL.Path, "/logs") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		if !api.validateAPIKey(r.Context(), apiKey) {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (api *API) validateAPIKey(ctx context.Context, apiKey string) bool {
	if api.config.JWTSecret == "change-me-in-production" {
		return apiKey == "dev-key"
	}

	hasher := sha256.New()
	hasher.Write([]byte(apiKey))
	keyHash := fmt.Sprintf("%x", hasher.Sum(nil))

	var count int
	err := api.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL",
		keyHash).Scan(&count)

	if err != nil {
		log.Printf("Failed to validate API key: %v", err)
		return false
	}

	if count > 0 {
		_, err := api.db.ExecContext(ctx,
			"UPDATE api_keys SET last_used_at = NOW() WHERE key_hash = $1",
			keyHash)
		if err != nil {
			log.Printf("Failed to update last_used_at: %v", err)
		}
		return true
	}

	return false
}

type rateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	requests := rl.requests[key]

	var validRequests []time.Time
	for _, req := range requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}

	if len(validRequests) >= rl.limit {
		return false
	}

	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		cutoff := time.Now().Add(-rl.window)

		for key, requests := range rl.requests {
			var validRequests []time.Time
			for _, req := range requests {
				if req.After(cutoff) {
					validRequests = append(validRequests, req)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mutex.Unlock()
	}
}

var globalRateLimiter *rateLimiter

func rateLimitMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	if globalRateLimiter == nil {
		globalRateLimiter = newRateLimiter(cfg.APIRateLimit, cfg.APIRateLimitWindow)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = strings.Split(forwarded, ",")[0]
			}

			if !globalRateLimiter.allow(clientIP) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
