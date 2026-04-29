package httpapi

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Options struct {
	RateLimitEnabled   bool
	RateLimitPerMinute int
}

type requestIDKey struct{}

func normalizeOptions(opts ...Options) Options {
	if len(opts) == 0 {
		return Options{RateLimitEnabled: true, RateLimitPerMinute: 600}
	}
	options := opts[0]
	if options.RateLimitPerMinute <= 0 {
		options.RateLimitPerMinute = 600
	}

	return options
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withRequestLogging(next http.Handler) http.Handler {
	log := s.logger
	if log == nil {
		log = zap.NewNop()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		status := recorder.status
		fields := []zap.Field{
			zap.String("request_id", requestIDFromContext(r.Context())),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", status),
			zap.Int("bytes", recorder.bytes),
			zap.Duration("duration", time.Since(started)),
			zap.String("remote_ip", clientIP(r)),
		}
		if status >= 500 {
			log.Error("http request completed", fields...)
			return
		}
		log.Info("http request completed", fields...)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.options.RateLimitEnabled || s.limiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		allowed, retryAfter := s.limiter.allow(clientIP(r))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		s.metrics.observe(r.Method, recorder.status, time.Since(started))
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		ip, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(body)
	r.bytes += n
	return n, err
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	windows map[string]rateWindow
}

type rateWindow struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		limit:   limit,
		windows: make(map[string]rateWindow),
	}
}

func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	if l.limit <= 0 {
		return true, 0
	}
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.windows[key]
	if window.reset.IsZero() || !now.Before(window.reset) {
		window = rateWindow{reset: now.Add(time.Minute)}
	}
	if window.count >= l.limit {
		return false, time.Until(window.reset)
	}

	window.count++
	l.windows[key] = window
	if len(l.windows) > 1000 {
		for ip, candidate := range l.windows {
			if now.After(candidate.reset) {
				delete(l.windows, ip)
			}
		}
	}

	return true, 0
}

type requestMetrics struct {
	mu                 sync.Mutex
	startedAt          time.Time
	requestCount       int64
	errorCount         int64
	totalDurationNanos int64
	statuses           map[int]int64
	methods            map[string]int64
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{
		startedAt: time.Now().UTC(),
		statuses:  make(map[int]int64),
		methods:   make(map[string]int64),
	}
}

func (m *requestMetrics) observe(method string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCount++
	if status >= 500 {
		m.errorCount++
	}
	m.totalDurationNanos += duration.Nanoseconds()
	m.statuses[status]++
	m.methods[method]++
}

func (m *requestMetrics) snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	statuses := make(map[string]int64, len(m.statuses))
	for status, count := range m.statuses {
		statuses[strconv.Itoa(status)] = count
	}
	methods := make(map[string]int64, len(m.methods))
	for method, count := range m.methods {
		methods[method] = count
	}

	averageDurationMS := float64(0)
	if m.requestCount > 0 {
		averageDurationMS = float64(m.totalDurationNanos) / float64(m.requestCount) / float64(time.Millisecond)
	}

	return map[string]any{
		"started_at":          m.startedAt,
		"request_count":       m.requestCount,
		"error_count":         m.errorCount,
		"average_duration_ms": averageDurationMS,
		"statuses":            statuses,
		"methods":             methods,
	}
}
