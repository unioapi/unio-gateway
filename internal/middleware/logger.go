package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ThankCat/unio-api/internal/httpx"
)

// statusRecorder 记录 handler 写出的 HTTP 状态码，供请求日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader 记录第一次写出的 HTTP 状态码，并保持 net/http 的首次写入语义。
func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}

	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

// Flush 将流式响应的 flush 能力转发给底层 ResponseWriter。
func (r *statusRecorder) Flush() {
	f, ok := r.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	f.Flush()
}

// Logger 记录每个 HTTP 请求的基础信息，包括方法、路径、状态码、耗时和请求 ID。
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			logger.InfoContext(
				r.Context(),
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", httpx.RequestID(r.Context()),
			)
		})
	}
}
