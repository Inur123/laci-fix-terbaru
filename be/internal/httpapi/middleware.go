package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRequestBody = 16 << 20

type rateEntry struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{entries: make(map[string]rateEntry)}
}

func (l *rateLimiter) allow(key string, maximum int, window time.Duration) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) >= 10_000 {
		var oldestKey string
		var oldest time.Time
		for candidate, value := range l.entries {
			if !now.Before(value.resetAt) {
				delete(l.entries, candidate)
				continue
			}
			if oldestKey == "" || value.resetAt.Before(oldest) {
				oldestKey, oldest = candidate, value.resetAt
			}
		}
		if len(l.entries) >= 10_000 && oldestKey != "" {
			delete(l.entries, oldestKey)
		}
	}

	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.resetAt) {
		l.entries[key] = rateEntry{count: 1, resetAt: now.Add(window)}
		return true, window
	}
	if entry.count >= maximum {
		return false, time.Until(entry.resetAt)
	}
	entry.count++
	l.entries[key] = entry

	return true, time.Until(entry.resetAt)
}

func (a *API) rateLimit(maximum int, window time.Duration, scope ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := r.URL.Path
			if len(scope) > 0 {
				target = scope[0]
			}
			key := clientIP(r) + "|" + r.Method + "|" + target
			allowed, retryAfter := a.limits.allow(key, maximum, window)
			if !allowed {
				seconds := int(retryAfter.Seconds())
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak permintaan. Silakan coba lagi nanti")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *API) limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (a *API) verifyOrigin(next http.Handler) http.Handler {
	frontend, _ := url.Parse(a.cfg.FrontendURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		parsed, err := url.Parse(origin)
		if err != nil || frontend == nil || parsed.Scheme != frontend.Scheme || parsed.Host != frontend.Host {
			writeError(w, http.StatusForbidden, "INVALID_ORIGIN", "Origin permintaan tidak diizinkan")
			return
		}
		next.ServeHTTP(w, r)
	})
}
