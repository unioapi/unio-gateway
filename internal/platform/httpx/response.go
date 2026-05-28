package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ThankCat/unio-api/internal/platform/failure"
)

const (
	// ContentTypeJSON 是 JSON 响应使用的 Content-Type。
	ContentTypeJSON = "application/json"

	// ContentTypeSSE 是 SSE 流式响应使用的 Content-Type。
	ContentTypeSSE = "text/event-stream"
)

var (
	// ErrStreamingUnsupported 表示当前 ResponseWriter 不支持流式 flush。
	ErrStreamingUnsupported = errors.New("streaming unsupported")
)

// ErrorResponse 是 API 错误响应的外层结构。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody 描述 API 错误的 OpenAI-compatible 响应体。
type ErrorBody struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    string  `json:"code"`
}

// WriteJSON 将 v 以 JSON 格式写入响应，并设置 HTTP 状态码。
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return failure.Wrap(
			failure.CodeHTTPResponseWriteFailed,
			err,
			failure.WithMessage("write json response"),
		)
	}

	return nil
}

// WriteSSE 将一条 SSE 数据写入响应，并立即 flush 给客户端。
func WriteSSE(w http.ResponseWriter, data []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return failure.Wrap(
			failure.CodeHTTPStreamingUnsupported,
			ErrStreamingUnsupported,
			failure.WithMessage(ErrStreamingUnsupported.Error()),
		)
	}

	header := w.Header()
	header.Set("Content-Type", ContentTypeSSE)
	header.Set("Cache-Control", "no-cache")
	header.Set("X-Accel-Buffering", "no")

	// TODO(阶段8/production): [GAP-8-002] HTTP SSE 写出当前只有 data-only helper，尚未抽象 event/id/retry/heartbeat 和写出后错误事件；阶段 8 stream observability 或新增 SSE endpoint 时；抽出项目级 SSE Writer 并覆盖多行 data、event/error、heartbeat、flush/context 失败。
	// SSE 的一条事件以空行结束；OpenAI-compatible stream 常用 data 行承载 JSON。
	if _, err := w.Write([]byte("data: ")); err != nil {
		return failure.Wrap(
			failure.CodeHTTPResponseWriteFailed,
			err,
			failure.WithMessage("write sse response"),
		)
	}

	if _, err := w.Write(data); err != nil {
		return failure.Wrap(
			failure.CodeHTTPResponseWriteFailed,
			err,
			failure.WithMessage("write sse response"),
		)
	}

	if _, err := w.Write([]byte("\n\n")); err != nil {
		return failure.Wrap(
			failure.CodeHTTPResponseWriteFailed,
			err,
			failure.WithMessage("write sse response"),
		)
	}

	flusher.Flush()
	return nil
}

// WriteError 写入统一格式的 JSON 错误响应。
func WriteError(w http.ResponseWriter, status int, code string, message string) error {
	return WriteOpenAIError(w, status, code, message, "api_error", nil)
}

// WriteOpenAIError 写入 OpenAI-compatible JSON 错误响应。
func WriteOpenAIError(w http.ResponseWriter, status int, code string, message string, errorType string, param *string) error {
	errBody := ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Type:    errorType,
			Param:   param,
		},
	}

	return WriteJSON(w, status, errBody)
}
