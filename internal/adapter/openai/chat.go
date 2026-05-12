package openai

import (
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
		Usage: adapter.ChatUsage{
			PromptTokens:     upstreamRespBody.Usage.PromptTokens,
			CompletionTokens: upstreamRespBody.Usage.CompletionTokens,
			TotalTokens:      upstreamRespBody.Usage.TotalTokens,
		},
	}, nil
}
