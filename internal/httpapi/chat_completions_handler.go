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

	// TODO(阶段4/production): [GAP-4-001] chat messages 目前只校验非空列表，未校验 role 合法性、content 空值策略和 stop/user 等字段边界；开放 OpenAI-compatible API 前；补齐请求 DTO 深度校验并保持 OpenAI-compatible 错误格式。
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
