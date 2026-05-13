package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ThankCat/unio-api/internal/httpx"
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
		_ = httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if req.Model == "" {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"model is required",
			"invalid_request_error",
			stringPtr("model"),
		)
		return
	}

	if len(req.Messages) == 0 {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"messages is required",
			"invalid_request_error",
			stringPtr("messages"),
		)
		return
	}

	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"temperature must be between 0 and 2",
			"invalid_request_error",
			stringPtr("temperature"),
		)
		return
	}

	if req.TopP != nil && (*req.TopP < 0 || *req.TopP > 1) {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"top_p must be between 0 and 1",
			"invalid_request_error",
			stringPtr("top_p"),
		)
		return
	}

	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		_ = httpx.WriteOpenAIError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"max_tokens must be greater than 0",
			"invalid_request_error",
			stringPtr("max_tokens"),
		)
		return
	}

	if req.Stream != nil && *req.Stream {
		err := h.service.StreamChatCompletion(r.Context(), req, func(chunk ChatCompletionStreamResponse) error {
			payload, err := json.Marshal(chunk)
			if err != nil {
				return err
			}

			return httpx.WriteSSE(w, payload)
		})
		if err != nil {
			// TODO(阶段5/production): stream 已写出部分 chunk 后不能再返回普通 JSON 错误；接入 stream error mapping 后改为 SSE 错误事件或中断连接并记录请求状态。
			_ = httpx.WriteError(w, http.StatusInternalServerError, "stream_chat_completion_error", err.Error())
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
