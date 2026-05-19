package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/ThankCat/unio-api/internal/httpx"
)

// RequestID 为每个请求补充请求 ID，并写入响应 header。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(httpx.HeaderRequestID)
		if requestID == "" {
			requestID = newRequestID()
		}

		// TODO(阶段1/production): [GAP-1-002] 直接信任客户端 X-Request-ID 会导致超长值或控制字符进入响应头和日志；开放公网 API 前；限制长度/字符集，非法时生成服务端 correlation id。
		w.Header().Set(httpx.HeaderRequestID, requestID)

		ctx := httpx.ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID 生成 16 字节随机请求 ID；随机源失败时返回兜底值。
func newRequestID() string {
	var b [16]byte

	if _, err := rand.Read(b[:16]); err != nil {
		return "unknown"
	}

	return hex.EncodeToString(b[:])
}
