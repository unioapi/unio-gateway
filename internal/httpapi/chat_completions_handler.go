package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ThankCat/unio-api/internal/httpx"
)

const (
	// maxStopSequences 是 OpenAI-compatible stop 序列数量上限。
	maxStopSequences = 4

	// maxUserLength 是 user 字段的保守长度上限，避免审计/风控标识无限膨胀。
	maxUserLength = 512
)

// ChatCompletionService 定义 chat completions handler 依赖的业务能力。
type ChatCompletionService interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)
	StreamChatCompletion(ctx context.Context, req ChatCompletionRequest, emit func(ChatCompletionStreamResponse) error) error
}

// chatCompletionsHandler 处理 OpenAI-compatible chat completions 请求。
type chatCompletionsHandler struct {
	service ChatCompletionService
}

// ServeHTTP 解析请求、调用 service，并写出 HTTP 响应。
func (h *chatCompletionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req ChatCompletionRequest

	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	if validationErr := validateChatCompletionRequest(req); validationErr != nil {
		writeChatValidationError(w, validationErr)
		return
	}

	if req.Stream != nil && *req.Stream {
		if _, ok := w.(http.Flusher); !ok {
			_ = httpx.WriteError(w, http.StatusInternalServerError, "streaming_unsupported", "streaming unsupported")
			return
		}

		streamStarted := false

		err := h.service.StreamChatCompletion(r.Context(), req, func(chunk ChatCompletionStreamResponse) error {
			payload, err := json.Marshal(chunk)
			if err != nil {
				return err
			}

			// 从这里开始，响应已经进入 SSE 写出路径；后续不能再退回普通 JSON error。
			streamStarted = true
			return httpx.WriteSSE(w, payload)
		})
		if err != nil {
			if streamStarted {
				// TODO(阶段7/production): [GAP-7-006] SSE 写出后无法再返回 OpenAI-compatible JSON error，客户端只能看到中断 stream；公开生产前；保留 request 状态和 final usage settlement 作为账务事实，并在后续错误事件/观测能力中暴露中断原因。
				return
			}

			_ = httpx.WriteOpenAIError(
				w,
				http.StatusInternalServerError,
				"stream_chat_completion_error",
				"stream chat completion failed",
				"api_error",
				nil,
			)
			return
		}

		_ = httpx.WriteSSE(w, []byte("[DONE]"))
		return
	}

	resp, err := h.service.CreateChatCompletion(r.Context(), req)
	if err != nil {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"chat completion failed",
			"api_error",
			nil,
		)
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)

}

// stringPtr 返回字符串指针，用于构造 optional 字段。
func stringPtr(v string) *string {
	return &v
}

// writeJSONDecodeError 将 JSON decode 错误转换成 OpenAI-compatible 错误响应。
func writeJSONDecodeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := "invalid json body"

	switch {
	case errors.Is(err, httpx.ErrUnsupportedContentType):
		status = http.StatusUnsupportedMediaType
		message = "content type must be application/json"
	case errors.Is(err, httpx.ErrRequestBodyTooLarge):
		status = http.StatusRequestEntityTooLarge
		message = "request body too large"
	case errors.Is(err, httpx.ErrEmptyJSONBody):
		message = "request body is required"
	case errors.Is(err, httpx.ErrTrailingJSONToken):
		message = "request body must contain a single JSON object"
	}

	_ = httpx.WriteOpenAIError(
		w,
		status,
		"invalid_request",
		message,
		"invalid_request_error",
		nil,
	)
}

// chatValidationError 表示 chat request validation 失败后的用户可见错误。
type chatValidationError struct {
	param   string
	message string
}

// validateChatCompletionRequest 校验 OpenAI-compatible chat request 的 HTTP DTO 边界。
func validateChatCompletionRequest(req ChatCompletionRequest) *chatValidationError {
	if strings.TrimSpace(req.Model) == "" {
		return &chatValidationError{param: "model", message: "model is required"}
	}

	if len(req.Messages) == 0 {
		return &chatValidationError{param: "messages", message: "messages is required"}
	}

	for i, msg := range req.Messages {
		roleParam := fmt.Sprintf("messages.%d.role", i)
		if strings.TrimSpace(msg.Role) == "" {
			return &chatValidationError{param: roleParam, message: "message role is required"}
		}

		if !isSupportedChatRole(msg.Role) {
			return &chatValidationError{
				param:   roleParam,
				message: "message role must be one of system, user, assistant",
			}
		}

		if strings.TrimSpace(msg.Content) == "" {
			return &chatValidationError{
				param:   fmt.Sprintf("messages.%d.content", i),
				message: "message content is required",
			}
		}
	}

	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return &chatValidationError{param: "temperature", message: "temperature must be between 0 and 2"}
	}

	if req.TopP != nil && (*req.TopP < 0 || *req.TopP > 1) {
		return &chatValidationError{param: "top_p", message: "top_p must be between 0 and 1"}
	}

	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return &chatValidationError{param: "max_tokens", message: "max_tokens must be greater than 0"}
	}

	if req.PresencePenalty != nil && (*req.PresencePenalty < -2 || *req.PresencePenalty > 2) {
		return &chatValidationError{param: "presence_penalty", message: "presence_penalty must be between -2 and 2"}
	}

	if req.FrequencyPenalty != nil && (*req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2) {
		return &chatValidationError{param: "frequency_penalty", message: "frequency_penalty must be between -2 and 2"}
	}

	if len(req.Stop) > maxStopSequences {
		return &chatValidationError{param: "stop", message: "stop must contain at most 4 sequences"}
	}

	for i, stop := range req.Stop {
		if strings.TrimSpace(stop) == "" {
			return &chatValidationError{
				param:   fmt.Sprintf("stop.%d", i),
				message: "stop sequence must not be empty",
			}
		}
	}

	if req.User != nil {
		if strings.TrimSpace(*req.User) == "" {
			return &chatValidationError{param: "user", message: "user must not be empty"}
		}

		if len(*req.User) > maxUserLength {
			return &chatValidationError{param: "user", message: "user must be at most 512 characters"}
		}
	}

	return nil
}

// isSupportedChatRole 判断当前 text-only MVP 支持的 chat role。
func isSupportedChatRole(role string) bool {
	switch role {
	case "system", "user", "assistant":
		return true
	default:
		return false
	}
}

// writeChatValidationError 将 chat validation 错误写成 OpenAI-compatible error。
func writeChatValidationError(w http.ResponseWriter, validationErr *chatValidationError) {
	_ = httpx.WriteOpenAIError(
		w,
		http.StatusBadRequest,
		"invalid_request",
		validationErr.message,
		"invalid_request_error",
		stringPtr(validationErr.param),
	)
}
