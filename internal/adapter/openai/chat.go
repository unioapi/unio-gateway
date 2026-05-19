package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
)

// Adapter 调用 OpenAI-compatible 上游接口。
type Adapter struct {
	client *http.Client
}

// NewAdapter 创建 OpenAI-compatible adapter。
func NewAdapter(client *http.Client) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}

	return &Adapter{client: client}
}

// ChatCompletions 调用上游 /chat/completions，并转换为统一 adapter 响应。
func (a *Adapter) ChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	if ch.BaseURL == "" {
		return nil, fmt.Errorf("openai adapter: channel base url is empty")
	}

	if ch.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ch.Timeout)
		defer cancel()
	}

	url := strings.TrimRight(ch.BaseURL, "/") + "/chat/completions"

	messages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	upstreamReqBody := chatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(upstreamReqBody); err != nil {
		return nil, fmt.Errorf("openai adapter: encode chat completion request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return nil, fmt.Errorf("openai adapter: create chat completion request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ch.APIKey))

	upstreamResp, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("openai adapter: send chat completion request: %w", err)
	}
	defer upstreamResp.Body.Close()

	if upstreamResp.StatusCode < http.StatusOK || upstreamResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openai adapter: upstream status %d", upstreamResp.StatusCode)
	}

	var upstreamRespBody chatCompletionResponse
	if err := json.NewDecoder(upstreamResp.Body).Decode(&upstreamRespBody); err != nil {
		return nil, fmt.Errorf("openai adapter: decode chat completion response: %w", err)
	}

	if upstreamRespBody.Choices == nil || len(upstreamRespBody.Choices) == 0 {
		return nil, fmt.Errorf("openai adapter: empty chat completion choices")
	}

	return &adapter.ChatResponse{
		ID:      upstreamRespBody.ID,
		Model:   upstreamRespBody.Model,
		Content: upstreamRespBody.Choices[0].Message.Content,
		Usage:   chatUsageFromOpenAI(upstreamRespBody.Usage),
	}, nil
}

// StreamChatCompletions 调用上游 /chat/completions stream，并转换为统一 adapter chunk。
func (a *Adapter) StreamChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest, emit func(adapter.ChatStreamChunk) error) error {
	if emit == nil {
		return fmt.Errorf("openai adapter: stream emit is nil")
	}

	if ch.BaseURL == "" {
		return fmt.Errorf("openai adapter: channel base url is empty")
	}

	if ch.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ch.Timeout)
		defer cancel()
	}

	url := strings.TrimRight(ch.BaseURL, "/") + "/chat/completions"

	messages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	upstreamReqBody := chatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   true,
		StreamOptions: &chatStreamOptions{
			IncludeUsage: true,
		},
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(upstreamReqBody); err != nil {
		return fmt.Errorf("openai adapter: encode stream chat completion request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return fmt.Errorf("openai adapter: create stream chat completion request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ch.APIKey))

	upstreamResp, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("openai adapter: send stream chat completion request: %w", err)
	}
	defer upstreamResp.Body.Close()

	if upstreamResp.StatusCode < http.StatusOK || upstreamResp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("openai adapter: upstream stream status %d", upstreamResp.StatusCode)
	}

	scanner := bufio.NewScanner(upstreamResp.Body)
	// TODO(阶段5/production): [GAP-5-002] bufio.Scanner 仍受单个 SSE event 大小上限影响，遇到超长 delta/tool_calls 可能中断 stream；支持工具调用或大 chunk 上游前；改为基于 reader 的 SSE event parser，并显式处理 backpressure 和超限错误。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var streamResp chatCompletionStreamResponse
		if err := json.Unmarshal([]byte(payload), &streamResp); err != nil {
			return fmt.Errorf("openai adapter: decode stream chunk: %w", err)
		}

		if len(streamResp.Choices) == 0 {
			if streamResp.Usage != nil {
				usage := chatUsageFromOpenAI(*streamResp.Usage)

				if err := emit(adapter.ChatStreamChunk{
					ID:    streamResp.ID,
					Model: streamResp.Model,
					Usage: &usage,
				}); err != nil {
					return fmt.Errorf("openai adapter: send stream usage chunk: %w", err)
				}
			}

			continue
		}

		choice := streamResp.Choices[0]

		// 上游可能发送空 delta 作为 stream 心跳或占位事件；这类 chunk 没有用户可见内容，
		// 也不携带结束原因，直接跳过，避免污染下游 SSE。
		if choice.Delta.Role == "" && choice.Delta.Content == "" && choice.FinishReason == nil {
			continue
		}

		chunk := adapter.ChatStreamChunk{
			ID:           streamResp.ID,
			Model:        streamResp.Model,
			Role:         choice.Delta.Role,
			Content:      choice.Delta.Content,
			FinishReason: choice.FinishReason,
		}

		if err := emit(chunk); err != nil {
			return fmt.Errorf("openai adapter: send stream chunk: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("openai adapter: scan stream chunk: %w", err)
	}

	return nil
}

// chatUsageFromOpenAI 将 OpenAI usage DTO 转成 adapter 内部 usage DTO。
func chatUsageFromOpenAI(usage chatCompletionUsage) adapter.ChatUsage {
	return adapter.ChatUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CachedTokens:     usage.PromptTokensDetails.CachedTokens,
		ReasoningTokens:  usage.CompletionTokensDetails.ReasoningTokens,
	}
}
