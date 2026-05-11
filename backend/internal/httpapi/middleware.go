package httpapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
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
	RateLimitBackend   DistributedRateLimiter
	CORSAllowedOrigins []string
	TrustedProxyCIDRs  []string

	allowedOrigins   map[string]struct{}
	trustedProxyNets []*net.IPNet
}

type DistributedRateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error)
}

type requestIDKey struct{}

func normalizeOptions(opts ...Options) Options {
	options := Options{RateLimitEnabled: true, RateLimitPerMinute: 600}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.RateLimitPerMinute <= 0 {
		options.RateLimitPerMinute = 600
	}
	if len(options.CORSAllowedOrigins) == 0 {
		options.CORSAllowedOrigins = []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
		}
	}
	options.allowedOrigins = normalizeAllowedOrigins(options.CORSAllowedOrigins)
	options.trustedProxyNets = normalizeTrustedProxyCIDRs(options.TrustedProxyCIDRs)

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
			zap.String("path", safeLogPath(r.URL.Path)),
			zap.Int("status", status),
			zap.Int("bytes", recorder.bytes),
			zap.Duration("duration", time.Since(started)),
			zap.String("remote_ip", s.clientIP(r)),
		}
		if status >= 500 {
			log.Error("http request completed", fields...)
			return
		}
		log.Info("http request completed", fields...)
	})
}

func (s *Server) withRecovery(next http.Handler) http.Handler {
	log := s.logger
	if log == nil {
		log = zap.NewNop()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("http panic recovered",
					zap.String("request_id", requestIDFromContext(r.Context())),
					zap.String("method", r.Method),
					zap.String("path", safeLogPath(r.URL.Path)),
					zap.String("remote_ip", s.clientIP(r)),
					zap.String("panic_type", fmt.Sprintf("%T", recovered)),
					zap.Stack("stack"),
				)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.options.RateLimitEnabled || s.limiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		key := s.clientIP(r)
		allowed, retryAfter, err := s.allowRateLimit(r.Context(), key)
		if err != nil && s.logger != nil {
			s.logger.Warn("distributed rate limit failed; using local fallback",
				zap.String("request_id", requestIDFromContext(r.Context())),
				zap.String("remote_ip", key),
				zap.Error(err),
			)
		}
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowRateLimit(ctx context.Context, key string) (bool, time.Duration, error) {
	if s.options.RateLimitBackend != nil {
		allowed, retryAfter, err := s.options.RateLimitBackend.Allow(ctx, key, s.options.RateLimitPerMinute, time.Minute)
		if err == nil {
			return allowed, retryAfter, nil
		}
		fallbackAllowed, fallbackRetryAfter := s.limiter.allow(key)
		return fallbackAllowed, fallbackRetryAfter, err
	}

	allowed, retryAfter := s.limiter.allow(key)
	return allowed, retryAfter, nil
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

func (s *Server) clientIP(r *http.Request) string {
	host := remoteIP(r)
	if host == "" {
		return strings.TrimSpace(r.RemoteAddr)
	}
	if !s.isTrustedProxy(host) {
		return host
	}

	chain := forwardedChain(r, host)
	for i := len(chain) - 1; i >= 0; i-- {
		ip := net.ParseIP(chain[i])
		if ip == nil {
			continue
		}
		if !s.isTrustedProxyIP(ip) {
			return ip.String()
		}
	}

	return host
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func forwardedChain(r *http.Request, remote string) []string {
	chain := make([]string, 0, 4)
	for _, header := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(header, ",") {
			if ip := strings.TrimSpace(part); ip != "" {
				chain = append(chain, ip)
			}
		}
	}
	chain = append(chain, remote)
	return chain
}

func (s *Server) isTrustedProxy(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	return s.isTrustedProxyIP(parsed)
}

func (s *Server) isTrustedProxyIP(ip net.IP) bool {
	for _, network := range s.options.trustedProxyNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func normalizeTrustedProxyCIDRs(values []string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if ip := net.ParseIP(value); ip != nil {
			networks = append(networks, exactIPNet(ip))
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err == nil && network != nil {
			networks = append(networks, network)
		}
	}
	return networks
}

func (s *Server) allowedCORSOrigin(origin string) string {
	normalized := normalizeOrigin(origin)
	if normalized == "" {
		return ""
	}
	if _, ok := s.options.allowedOrigins[normalized]; ok {
		return normalized
	}
	return ""
}

func normalizeAllowedOrigins(values []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		origin := normalizeOrigin(value)
		if origin == "" || origin == "*" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return allowed
}

func normalizeOrigin(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return ""
	}

	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}

func exactIPNet(ip net.IP) *net.IPNet {
	if v4 := ip.To4(); v4 != nil {
		return &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	body   []byte
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
	if len(r.body) < maxRecordedResponseBodyBytes {
		remaining := maxRecordedResponseBodyBytes - len(r.body)
		if len(body) > remaining {
			body = body[:remaining]
		}
		r.body = append(r.body, body...)
	}
	return n, err
}

func (r *statusRecorder) Body() []byte {
	if len(r.body) == 0 {
		return nil
	}
	return append([]byte(nil), r.body...)
}

const maxRecordedResponseBodyBytes = 64 << 10

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
