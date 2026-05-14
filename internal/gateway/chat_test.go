package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/channel"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/routing"
)

// fakeChatRouter 是 gateway 测试使用的 routing 替身。
type fakeChatRouter struct {
	called bool
	req    routing.ChatRouteRequest
	plan   routing.ChatRoutePlan
	err    error
}

// PlanChat 记录 gateway 传入的 routing 请求，并返回测试预设计划。
func (r *fakeChatRouter) PlanChat(ctx context.Context, req routing.ChatRouteRequest) (routing.ChatRoutePlan, error) {
	r.called = true
	r.req = req
	return r.plan, r.err
}

// fakeAdapterRegistry 是 gateway 测试使用的 adapter registry 替身。
type fakeAdapterRegistry struct {
	chatKeys           []string
	streamChatKeys     []string
	chatAdapters       map[string]adapter.ChatAdapter
	streamChatAdapters map[string]adapter.StreamChatAdapter
}

// Chat 记录 adapter key，并按 key 返回测试预设非流式 adapter。
func (r *fakeAdapterRegistry) Chat(adapterKey string) (adapter.ChatAdapter, bool) {
	r.chatKeys = append(r.chatKeys, adapterKey)
	chatAdapter, ok := r.chatAdapters[adapterKey]
	return chatAdapter, ok
}

// StreamChat 记录 adapter key，并按 key 返回测试预设流式 adapter。
func (r *fakeAdapterRegistry) StreamChat(adapterKey string) (adapter.StreamChatAdapter, bool) {
	r.streamChatKeys = append(r.streamChatKeys, adapterKey)
	streamChatAdapter, ok := r.streamChatAdapters[adapterKey]
	return streamChatAdapter, ok
}

// fakeRetryClassifier 是 gateway 测试使用的 retry 判断替身。
type fakeRetryClassifier struct {
	retryable bool
	called    int
}

// IsRetryable 记录调用次数，并返回测试预设的 retry 判断结果。
func (c *fakeRetryClassifier) IsRetryable(err error) bool {
	c.called++
	return c.retryable
}

// fakeChatAdapter 是 gateway 测试使用的 adapter 替身。
type fakeChatAdapter struct {
	chatCalled   int
	chatReq      adapter.ChatRequest
	chatResp     *adapter.ChatResponse
	chatErr      error
	streamCalled int
	streamReq    adapter.ChatRequest
	streamResp   []adapter.ChatStreamChunk
	streamErr    error
	ch           channel.Runtime
}

// ChatCompletions 记录 gateway 传入的请求，并返回测试预设响应。
func (a *fakeChatAdapter) ChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	a.chatCalled++
	a.chatReq = req
	a.ch = ch

	return a.chatResp, a.chatErr
}

// StreamChatCompletions 记录 gateway 传入的流式请求，并逐个发出测试预设 chunk。
func (a *fakeChatAdapter) StreamChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest, emit func(adapter.ChatStreamChunk) error) error {
	a.streamCalled++
	a.streamReq = req
	a.ch = ch

	if a.streamErr != nil && len(a.streamResp) == 0 {
		return a.streamErr
	}

	for _, chunk := range a.streamResp {
		if err := emit(chunk); err != nil {
			return err
		}
	}

	return a.streamErr
}

// contextWithPrincipal 创建带 API key principal 的测试 context。
func contextWithPrincipal(projectID int64) context.Context {
	return auth.ContextWithAPIKeyPrincipal(context.Background(), &auth.APIKeyPrincipal{
		APIKeyID:  1,
		ProjectID: projectID,
		KeyPrefix: "unio_sk_test",
	})
}

// chatRequest 创建测试用 HTTP chat completion 请求。
func chatRequest() httpapi.ChatCompletionRequest {
	return httpapi.ChatCompletionRequest{
		Model:    "openai/gpt-4.1",
		Messages: []httpapi.ChatMessage{{Role: "user", Content: "hello"}},
	}
}

// routePlan 创建测试用同模型 route plan。
func routePlan(candidates ...routing.ChatRouteCandidate) routing.ChatRoutePlan {
	return routing.ChatRoutePlan{
		RequestedModel: "openai/gpt-4.1",
		Candidates:     candidates,
	}
}

// routeCandidate 创建测试用 route candidate。
func routeCandidate(adapterKey string, channelID int64, upstreamModel string) routing.ChatRouteCandidate {
	return routing.ChatRouteCandidate{
		AdapterKey: adapterKey,
		Channel: channel.Runtime{
			ID:      channelID,
			BaseURL: "https://example.test/v1",
			APIKey:  "test-secret",
			Timeout: 30 * time.Second,
		},
		UpstreamModel: upstreamModel,
	}
}

// chatResponse 创建测试用 adapter 响应。
func chatResponse(content string) *adapter.ChatResponse {
	return &adapter.ChatResponse{
		ID:      "chatcmpl_provider_test",
		Model:   "gpt-4.1",
		Content: content,
		Usage: adapter.ChatUsage{
			PromptTokens:     10,
			CompletionTokens: 11,
			TotalTokens:      21,
		},
	}
}

func TestChatCompletionServiceCreateChatCompletionRoutesAndCallsAdapter(t *testing.T) {
	fakeAdapter := &fakeChatAdapter{
		chatResp: chatResponse("adapter response"),
	}
	router := &fakeChatRouter{
		plan: routePlan(routeCandidate("openai", 123, "gpt-4.1")),
	}
	registry := &fakeAdapterRegistry{
		chatAdapters: map[string]adapter.ChatAdapter{
			"openai": fakeAdapter,
		},
	}
	service := NewChatCompletionService(router, registry, nil)

	got, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if err != nil {
		t.Fatalf("CreateChatCompletion returned err: %v", err)
	}

	if !router.called {
		t.Fatal("expected routing to be called")
	}
	if router.req.ProjectID != 42 {
		t.Fatalf("expected project id %d, got %d", int64(42), router.req.ProjectID)
	}
	if router.req.ModelID != "openai/gpt-4.1" {
		t.Fatalf("expected requested model %q, got %q", "openai/gpt-4.1", router.req.ModelID)
	}
	if len(registry.chatKeys) != 1 || registry.chatKeys[0] != "openai" {
		t.Fatalf("expected adapter key %q, got %#v", "openai", registry.chatKeys)
	}
	if fakeAdapter.chatCalled != 1 {
		t.Fatalf("expected adapter to be called once, got %d", fakeAdapter.chatCalled)
	}
	if fakeAdapter.chatReq.Model != "gpt-4.1" {
		t.Fatalf("expected upstream model %q, got %q", "gpt-4.1", fakeAdapter.chatReq.Model)
	}
	if fakeAdapter.ch.ID != 123 {
		t.Fatalf("expected channel id %d, got %d", int64(123), fakeAdapter.ch.ID)
	}
	if fakeAdapter.chatReq.Messages[0].Content != "hello" {
		t.Fatalf("expected message content %q, got %q", "hello", fakeAdapter.chatReq.Messages[0].Content)
	}
	if got.Model != "openai/gpt-4.1" {
		t.Fatalf("expected response model %q, got %q", "openai/gpt-4.1", got.Model)
	}
	if got.Choices[0].Message.Content != "adapter response" {
		t.Fatalf("expected content %q, got %q", "adapter response", got.Choices[0].Message.Content)
	}
	if got.Usage.TotalTokens != 21 {
		t.Fatalf("expected total_tokens %d, got %d", 21, got.Usage.TotalTokens)
	}
}

func TestChatCompletionServiceCreateChatCompletionDoesNotCallAdapterOnRoutingError(t *testing.T) {
	routingErr := errors.New("no route")
	fakeAdapter := &fakeChatAdapter{}
	service := NewChatCompletionService(
		&fakeChatRouter{err: routingErr},
		&fakeAdapterRegistry{
			chatAdapters: map[string]adapter.ChatAdapter{
				"openai": fakeAdapter,
			},
		},
		nil,
	)

	_, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if !errors.Is(err, routingErr) {
		t.Fatalf("expected routing error, got %v", err)
	}
	if fakeAdapter.chatCalled != 0 {
		t.Fatalf("expected adapter not to be called, got %d calls", fakeAdapter.chatCalled)
	}
}

func TestChatCompletionServiceCreateChatCompletionReturnsMissingAdapterWithoutRetry(t *testing.T) {
	classifier := &fakeRetryClassifier{retryable: true}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(routeCandidate("missing", 123, "gpt-4.1"))},
		&fakeAdapterRegistry{chatAdapters: map[string]adapter.ChatAdapter{}},
		classifier,
	)

	_, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if err == nil {
		t.Fatal("expected missing adapter error")
	}
	if classifier.called != 0 {
		t.Fatalf("expected retry classifier not to be called, got %d calls", classifier.called)
	}
}

func TestChatCompletionServiceCreateChatCompletionFallsBackOnRetryableAdapterError(t *testing.T) {
	upstreamErr := errors.New("temporary upstream error")
	firstAdapter := &fakeChatAdapter{chatErr: upstreamErr}
	secondAdapter := &fakeChatAdapter{chatResp: chatResponse("fallback response")}
	classifier := &fakeRetryClassifier{retryable: true}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(
			routeCandidate("openai-primary", 101, "gpt-4.1"),
			routeCandidate("openai-secondary", 102, "gpt-4.1"),
		)},
		&fakeAdapterRegistry{
			chatAdapters: map[string]adapter.ChatAdapter{
				"openai-primary":   firstAdapter,
				"openai-secondary": secondAdapter,
			},
		},
		classifier,
	)

	got, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if err != nil {
		t.Fatalf("CreateChatCompletion returned err: %v", err)
	}
	if firstAdapter.chatCalled != 1 {
		t.Fatalf("expected first adapter to be called once, got %d", firstAdapter.chatCalled)
	}
	if secondAdapter.chatCalled != 1 {
		t.Fatalf("expected second adapter to be called once, got %d", secondAdapter.chatCalled)
	}
	if classifier.called != 1 {
		t.Fatalf("expected retry classifier to be called once, got %d", classifier.called)
	}
	if got.Choices[0].Message.Content != "fallback response" {
		t.Fatalf("expected fallback response, got %q", got.Choices[0].Message.Content)
	}
}

func TestChatCompletionServiceCreateChatCompletionDoesNotFallbackOnNonRetryableAdapterError(t *testing.T) {
	upstreamErr := errors.New("invalid upstream request")
	firstAdapter := &fakeChatAdapter{chatErr: upstreamErr}
	secondAdapter := &fakeChatAdapter{chatResp: chatResponse("fallback response")}
	classifier := &fakeRetryClassifier{retryable: false}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(
			routeCandidate("openai-primary", 101, "gpt-4.1"),
			routeCandidate("openai-secondary", 102, "gpt-4.1"),
		)},
		&fakeAdapterRegistry{
			chatAdapters: map[string]adapter.ChatAdapter{
				"openai-primary":   firstAdapter,
				"openai-secondary": secondAdapter,
			},
		},
		classifier,
	)

	_, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("expected non-retryable adapter error, got %v", err)
	}
	if firstAdapter.chatCalled != 1 {
		t.Fatalf("expected first adapter to be called once, got %d", firstAdapter.chatCalled)
	}
	if secondAdapter.chatCalled != 0 {
		t.Fatalf("expected second adapter not to be called, got %d", secondAdapter.chatCalled)
	}
	if classifier.called != 1 {
		t.Fatalf("expected retry classifier to be called once, got %d", classifier.called)
	}
}

func TestChatCompletionServiceStreamChatCompletionRoutesAndCallsAdapter(t *testing.T) {
	fakeAdapter := &fakeChatAdapter{
		streamResp: []adapter.ChatStreamChunk{
			{
				ID:      "chatcmpl_mock",
				Model:   "gpt-4.1",
				Role:    "assistant",
				Content: "mock response",
			},
		},
	}
	router := &fakeChatRouter{
		plan: routePlan(routeCandidate("openai", 123, "gpt-4.1")),
	}
	registry := &fakeAdapterRegistry{
		streamChatAdapters: map[string]adapter.StreamChatAdapter{
			"openai": fakeAdapter,
		},
	}
	service := NewChatCompletionService(router, registry, nil)

	chunks := make([]httpapi.ChatCompletionStreamResponse, 0)
	err := service.StreamChatCompletion(contextWithPrincipal(42), chatRequest(), func(chunk httpapi.ChatCompletionStreamResponse) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChatCompletion returned err: %v", err)
	}

	if !router.called {
		t.Fatal("expected routing to be called")
	}
	if len(registry.streamChatKeys) != 1 || registry.streamChatKeys[0] != "openai" {
		t.Fatalf("expected adapter key %q, got %#v", "openai", registry.streamChatKeys)
	}
	if fakeAdapter.streamCalled != 1 {
		t.Fatalf("expected stream adapter to be called once, got %d", fakeAdapter.streamCalled)
	}
	if fakeAdapter.streamReq.Model != "gpt-4.1" {
		t.Fatalf("expected upstream model %q, got %q", "gpt-4.1", fakeAdapter.streamReq.Model)
	}
	if fakeAdapter.ch.ID != 123 {
		t.Fatalf("expected channel id %d, got %d", int64(123), fakeAdapter.ch.ID)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Model != "openai/gpt-4.1" {
		t.Fatalf("expected stream response model %q, got %q", "openai/gpt-4.1", chunks[0].Model)
	}
	if chunks[0].Choices[0].Delta.Content != "mock response" {
		t.Fatalf("expected content %q, got %q", "mock response", chunks[0].Choices[0].Delta.Content)
	}
}

func TestChatCompletionServiceStreamChatCompletionReturnsMissingAdapterWithoutRetry(t *testing.T) {
	classifier := &fakeRetryClassifier{retryable: true}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(routeCandidate("missing", 123, "gpt-4.1"))},
		&fakeAdapterRegistry{streamChatAdapters: map[string]adapter.StreamChatAdapter{}},
		classifier,
	)

	err := service.StreamChatCompletion(contextWithPrincipal(42), chatRequest(), func(chunk httpapi.ChatCompletionStreamResponse) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected missing stream adapter error")
	}
	if classifier.called != 0 {
		t.Fatalf("expected retry classifier not to be called, got %d calls", classifier.called)
	}
}

func TestChatCompletionServiceStreamChatCompletionFallsBackBeforeFirstChunk(t *testing.T) {
	upstreamErr := errors.New("temporary stream upstream error")
	firstAdapter := &fakeChatAdapter{streamErr: upstreamErr}
	secondAdapter := &fakeChatAdapter{
		streamResp: []adapter.ChatStreamChunk{
			{
				ID:      "chatcmpl_mock",
				Model:   "gpt-4.1",
				Role:    "assistant",
				Content: "fallback stream response",
			},
		},
	}
	classifier := &fakeRetryClassifier{retryable: true}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(
			routeCandidate("openai-primary", 101, "gpt-4.1"),
			routeCandidate("openai-secondary", 102, "gpt-4.1"),
		)},
		&fakeAdapterRegistry{
			streamChatAdapters: map[string]adapter.StreamChatAdapter{
				"openai-primary":   firstAdapter,
				"openai-secondary": secondAdapter,
			},
		},
		classifier,
	)

	chunks := make([]httpapi.ChatCompletionStreamResponse, 0)
	err := service.StreamChatCompletion(contextWithPrincipal(42), chatRequest(), func(chunk httpapi.ChatCompletionStreamResponse) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChatCompletion returned err: %v", err)
	}
	if firstAdapter.streamCalled != 1 {
		t.Fatalf("expected first stream adapter to be called once, got %d", firstAdapter.streamCalled)
	}
	if secondAdapter.streamCalled != 1 {
		t.Fatalf("expected second stream adapter to be called once, got %d", secondAdapter.streamCalled)
	}
	if classifier.called != 1 {
		t.Fatalf("expected retry classifier to be called once, got %d", classifier.called)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 stream chunk, got %d", len(chunks))
	}
	if chunks[0].Choices[0].Delta.Content != "fallback stream response" {
		t.Fatalf("expected fallback stream response, got %q", chunks[0].Choices[0].Delta.Content)
	}
}

func TestChatCompletionServiceStreamChatCompletionDoesNotFallbackAfterFirstChunk(t *testing.T) {
	upstreamErr := errors.New("stream failed after first chunk")
	firstAdapter := &fakeChatAdapter{
		streamResp: []adapter.ChatStreamChunk{
			{
				ID:      "chatcmpl_mock",
				Model:   "gpt-4.1",
				Role:    "assistant",
				Content: "partial",
			},
		},
		streamErr: upstreamErr,
	}
	secondAdapter := &fakeChatAdapter{
		streamResp: []adapter.ChatStreamChunk{
			{
				ID:      "chatcmpl_mock",
				Model:   "gpt-4.1",
				Role:    "assistant",
				Content: "should not emit",
			},
		},
	}
	classifier := &fakeRetryClassifier{retryable: true}
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(
			routeCandidate("openai-primary", 101, "gpt-4.1"),
			routeCandidate("openai-secondary", 102, "gpt-4.1"),
		)},
		&fakeAdapterRegistry{
			streamChatAdapters: map[string]adapter.StreamChatAdapter{
				"openai-primary":   firstAdapter,
				"openai-secondary": secondAdapter,
			},
		},
		classifier,
	)

	chunks := make([]httpapi.ChatCompletionStreamResponse, 0)
	err := service.StreamChatCompletion(contextWithPrincipal(42), chatRequest(), func(chunk httpapi.ChatCompletionStreamResponse) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("expected stream error after first chunk, got %v", err)
	}
	if firstAdapter.streamCalled != 1 {
		t.Fatalf("expected first stream adapter to be called once, got %d", firstAdapter.streamCalled)
	}
	if secondAdapter.streamCalled != 0 {
		t.Fatalf("expected second stream adapter not to be called, got %d", secondAdapter.streamCalled)
	}
	if classifier.called != 0 {
		t.Fatalf("expected retry classifier not to be called after emit, got %d", classifier.called)
	}
	if len(chunks) != 1 || chunks[0].Choices[0].Delta.Content != "partial" {
		t.Fatalf("expected only partial chunk, got %#v", chunks)
	}
}
