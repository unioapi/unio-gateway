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
	"github.com/ThankCat/unio-api/internal/httpx"
	"github.com/ThankCat/unio-api/internal/requestlog"
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

// fakeRequestLogService 是 gateway 测试使用的 requestlog 替身。
type fakeRequestLogService struct {
	nextRequestID int64
	nextAttemptID int64

	createRequests           []requestlog.CreateRequestParams
	markRequestRunningIDs    []int64
	markRequestSucceededArgs []requestlog.MarkRequestSucceededParams
	markRequestFailedArgs    []requestlog.MarkRequestFailedParams

	createAttempts           []requestlog.CreateAttemptParams
	markAttemptSucceededArgs []requestlog.MarkAttemptSucceededParams
	markAttemptFailedArgs    []requestlog.MarkAttemptFailedParams
}

// newFakeRequestLogService 创建测试用 requestlog.Service。
func newFakeRequestLogService() *fakeRequestLogService {
	return &fakeRequestLogService{
		nextRequestID: 1,
		nextAttemptID: 1,
	}
}

// CreateRequest 记录创建 request record 的参数并返回 pending 记录。
func (s *fakeRequestLogService) CreateRequest(ctx context.Context, params requestlog.CreateRequestParams) (requestlog.RequestRecord, error) {
	s.createRequests = append(s.createRequests, params)

	id := s.nextRequestID
	s.nextRequestID++

	return requestlog.RequestRecord{
		ID:               id,
		RequestID:        params.RequestID,
		UserID:           params.UserID,
		ProjectID:        params.ProjectID,
		APIKeyID:         params.APIKeyID,
		RequestedModelID: params.RequestedModelID,
		Stream:           params.Stream,
		Status:           requestlog.RequestStatusPending,
		StartedAt:        params.StartedAt,
	}, nil
}

// MarkRequestRunning 记录 request running 状态变更。
func (s *fakeRequestLogService) MarkRequestRunning(ctx context.Context, id int64) (requestlog.RequestRecord, error) {
	s.markRequestRunningIDs = append(s.markRequestRunningIDs, id)

	return requestlog.RequestRecord{
		ID:     id,
		Status: requestlog.RequestStatusRunning,
	}, nil
}

// MarkRequestSucceeded 记录 request succeeded 状态变更。
func (s *fakeRequestLogService) MarkRequestSucceeded(ctx context.Context, params requestlog.MarkRequestSucceededParams) (requestlog.RequestRecord, error) {
	s.markRequestSucceededArgs = append(s.markRequestSucceededArgs, params)

	return requestlog.RequestRecord{
		ID:     params.ID,
		Status: requestlog.RequestStatusSucceeded,
	}, nil
}

// MarkRequestFailed 记录 request failed 状态变更。
func (s *fakeRequestLogService) MarkRequestFailed(ctx context.Context, params requestlog.MarkRequestFailedParams) (requestlog.RequestRecord, error) {
	s.markRequestFailedArgs = append(s.markRequestFailedArgs, params)

	return requestlog.RequestRecord{
		ID:     params.ID,
		Status: requestlog.RequestStatusFailed,
	}, nil
}

// MarkRequestCanceled 记录 request canceled 状态变更。
func (s *fakeRequestLogService) MarkRequestCanceled(ctx context.Context, params requestlog.MarkRequestCanceledParams) (requestlog.RequestRecord, error) {
	return requestlog.RequestRecord{
		ID:     params.ID,
		Status: requestlog.RequestStatusCanceled,
	}, nil
}

// CreateAttempt 记录创建 request attempt 的参数并返回 running 记录。
func (s *fakeRequestLogService) CreateAttempt(ctx context.Context, params requestlog.CreateAttemptParams) (requestlog.AttemptRecord, error) {
	s.createAttempts = append(s.createAttempts, params)

	id := s.nextAttemptID
	s.nextAttemptID++

	return requestlog.AttemptRecord{
		ID:              id,
		RequestRecordID: params.RequestRecordID,
		AttemptIndex:    params.AttemptIndex,
		ProviderID:      params.ProviderID,
		ChannelID:       params.ChannelID,
		AdapterKey:      params.AdapterKey,
		UpstreamModel:   params.UpstreamModel,
		Status:          requestlog.AttemptStatusRunning,
		StartedAt:       params.StartedAt,
	}, nil
}

// MarkAttemptSucceeded 记录 attempt succeeded 状态变更。
func (s *fakeRequestLogService) MarkAttemptSucceeded(ctx context.Context, params requestlog.MarkAttemptSucceededParams) (requestlog.AttemptRecord, error) {
	s.markAttemptSucceededArgs = append(s.markAttemptSucceededArgs, params)

	return requestlog.AttemptRecord{
		ID:     params.ID,
		Status: requestlog.AttemptStatusSucceeded,
	}, nil
}

// MarkAttemptFailed 记录 attempt failed 状态变更。
func (s *fakeRequestLogService) MarkAttemptFailed(ctx context.Context, params requestlog.MarkAttemptFailedParams) (requestlog.AttemptRecord, error) {
	s.markAttemptFailedArgs = append(s.markAttemptFailedArgs, params)

	return requestlog.AttemptRecord{
		ID:     params.ID,
		Status: requestlog.AttemptStatusFailed,
	}, nil
}

// MarkAttemptCanceled 记录 attempt canceled 状态变更。
func (s *fakeRequestLogService) MarkAttemptCanceled(ctx context.Context, params requestlog.MarkAttemptCanceledParams) (requestlog.AttemptRecord, error) {
	return requestlog.AttemptRecord{
		ID:     params.ID,
		Status: requestlog.AttemptStatusCanceled,
	}, nil
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
	ctx := httpx.ContextWithRequestID(context.Background(), "gateway-test-request-id")

	return auth.ContextWithAPIKeyPrincipal(ctx, &auth.APIKeyPrincipal{
		APIKeyID:  1,
		UserID:    7,
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
		ProviderID: 9000 + channelID,
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
	requestLog := newFakeRequestLogService()
	service := NewChatCompletionService(router, registry, nil, requestLog)

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
	if len(requestLog.createRequests) != 1 {
		t.Fatalf("expected one request record, got %d", len(requestLog.createRequests))
	}
	if requestLog.createRequests[0].UserID != 7 {
		t.Fatalf("expected user id %d, got %d", int64(7), requestLog.createRequests[0].UserID)
	}
	if requestLog.createRequests[0].ProjectID != 42 {
		t.Fatalf("expected request project id %d, got %d", int64(42), requestLog.createRequests[0].ProjectID)
	}
	if requestLog.createRequests[0].RequestedModelID != "openai/gpt-4.1" {
		t.Fatalf("expected requested model %q, got %q", "openai/gpt-4.1", requestLog.createRequests[0].RequestedModelID)
	}
	if len(requestLog.markRequestRunningIDs) != 1 || requestLog.markRequestRunningIDs[0] != 1 {
		t.Fatalf("expected request to be marked running once, got %#v", requestLog.markRequestRunningIDs)
	}
	if len(requestLog.createAttempts) != 1 {
		t.Fatalf("expected one request attempt, got %d", len(requestLog.createAttempts))
	}
	if requestLog.createAttempts[0].ProviderID != 9123 {
		t.Fatalf("expected provider id %d, got %d", int64(9123), requestLog.createAttempts[0].ProviderID)
	}
	if requestLog.createAttempts[0].ChannelID != 123 {
		t.Fatalf("expected attempt channel id %d, got %d", int64(123), requestLog.createAttempts[0].ChannelID)
	}
	if len(requestLog.markAttemptSucceededArgs) != 1 {
		t.Fatalf("expected one succeeded attempt, got %d", len(requestLog.markAttemptSucceededArgs))
	}
	if requestLog.markAttemptSucceededArgs[0].UpstreamRequestID != nil {
		t.Fatalf("expected unknown upstream request id to stay nil, got %v", requestLog.markAttemptSucceededArgs[0].UpstreamRequestID)
	}
	if len(requestLog.markRequestSucceededArgs) != 1 {
		t.Fatalf("expected one succeeded request, got %d", len(requestLog.markRequestSucceededArgs))
	}
	if requestLog.markRequestSucceededArgs[0].FinalProviderID != 9123 {
		t.Fatalf("expected final provider id %d, got %d", int64(9123), requestLog.markRequestSucceededArgs[0].FinalProviderID)
	}
}

func TestChatCompletionServiceCreateChatCompletionDoesNotCallAdapterOnRoutingError(t *testing.T) {
	routingErr := errors.New("no route")
	fakeAdapter := &fakeChatAdapter{}
	requestLog := newFakeRequestLogService()
	service := NewChatCompletionService(
		&fakeChatRouter{err: routingErr},
		&fakeAdapterRegistry{
			chatAdapters: map[string]adapter.ChatAdapter{
				"openai": fakeAdapter,
			},
		},
		nil,
		requestLog,
	)

	_, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if !errors.Is(err, routingErr) {
		t.Fatalf("expected routing error, got %v", err)
	}
	if fakeAdapter.chatCalled != 0 {
		t.Fatalf("expected adapter not to be called, got %d calls", fakeAdapter.chatCalled)
	}
	if len(requestLog.createRequests) != 1 {
		t.Fatalf("expected one request record, got %d", len(requestLog.createRequests))
	}
	if len(requestLog.createAttempts) != 0 {
		t.Fatalf("expected no attempts on routing error, got %d", len(requestLog.createAttempts))
	}
	if len(requestLog.markRequestFailedArgs) != 1 {
		t.Fatalf("expected one failed request, got %d", len(requestLog.markRequestFailedArgs))
	}
	if requestLog.markRequestFailedArgs[0].ErrorCode != "routing_error" {
		t.Fatalf("expected routing_error, got %q", requestLog.markRequestFailedArgs[0].ErrorCode)
	}
}

func TestChatCompletionServiceCreateChatCompletionReturnsMissingAdapterWithoutRetry(t *testing.T) {
	classifier := &fakeRetryClassifier{retryable: true}
	requestLog := newFakeRequestLogService()
	service := NewChatCompletionService(
		&fakeChatRouter{plan: routePlan(routeCandidate("missing", 123, "gpt-4.1"))},
		&fakeAdapterRegistry{chatAdapters: map[string]adapter.ChatAdapter{}},
		classifier,
		requestLog,
	)

	_, err := service.CreateChatCompletion(contextWithPrincipal(42), chatRequest())
	if err == nil {
		t.Fatal("expected missing adapter error")
	}
	if classifier.called != 0 {
		t.Fatalf("expected retry classifier not to be called, got %d calls", classifier.called)
	}
	if len(requestLog.createAttempts) != 1 {
		t.Fatalf("expected one attempt for missing adapter, got %d", len(requestLog.createAttempts))
	}
	if len(requestLog.markAttemptFailedArgs) != 1 {
		t.Fatalf("expected one failed attempt, got %d", len(requestLog.markAttemptFailedArgs))
	}
	if requestLog.markAttemptFailedArgs[0].ErrorCode != "adapter_not_registered" {
		t.Fatalf("expected adapter_not_registered, got %q", requestLog.markAttemptFailedArgs[0].ErrorCode)
	}
	if requestLog.markAttemptFailedArgs[0].UpstreamStatusCode != nil {
		t.Fatalf("expected unknown upstream status to stay nil, got %v", requestLog.markAttemptFailedArgs[0].UpstreamStatusCode)
	}
	if len(requestLog.markRequestFailedArgs) != 1 {
		t.Fatalf("expected one failed request, got %d", len(requestLog.markRequestFailedArgs))
	}
	if requestLog.markRequestFailedArgs[0].ErrorCode != "adapter_not_registered" {
		t.Fatalf("expected adapter_not_registered, got %q", requestLog.markRequestFailedArgs[0].ErrorCode)
	}
}

func TestChatCompletionServiceCreateChatCompletionFallsBackOnRetryableAdapterError(t *testing.T) {
	upstreamErr := errors.New("temporary upstream error")
	firstAdapter := &fakeChatAdapter{chatErr: upstreamErr}
	secondAdapter := &fakeChatAdapter{chatResp: chatResponse("fallback response")}
	classifier := &fakeRetryClassifier{retryable: true}
	requestLog := newFakeRequestLogService()
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
		requestLog,
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
	if len(requestLog.createAttempts) != 2 {
		t.Fatalf("expected two attempts, got %d", len(requestLog.createAttempts))
	}
	if requestLog.createAttempts[0].AttemptIndex != 0 || requestLog.createAttempts[1].AttemptIndex != 1 {
		t.Fatalf("expected attempt indexes 0 and 1, got %#v", requestLog.createAttempts)
	}
	if len(requestLog.markAttemptFailedArgs) != 1 {
		t.Fatalf("expected first attempt to fail once, got %d", len(requestLog.markAttemptFailedArgs))
	}
	if requestLog.markAttemptFailedArgs[0].ErrorCode != "adapter_error" {
		t.Fatalf("expected adapter_error, got %q", requestLog.markAttemptFailedArgs[0].ErrorCode)
	}
	if len(requestLog.markAttemptSucceededArgs) != 1 {
		t.Fatalf("expected second attempt to succeed once, got %d", len(requestLog.markAttemptSucceededArgs))
	}
	if len(requestLog.markRequestSucceededArgs) != 1 {
		t.Fatalf("expected request to succeed once, got %d", len(requestLog.markRequestSucceededArgs))
	}
	if len(requestLog.markRequestFailedArgs) != 0 {
		t.Fatalf("expected request not to fail, got %#v", requestLog.markRequestFailedArgs)
	}
}

func TestChatCompletionServiceCreateChatCompletionDoesNotFallbackOnNonRetryableAdapterError(t *testing.T) {
	upstreamErr := errors.New("invalid upstream request")
	firstAdapter := &fakeChatAdapter{chatErr: upstreamErr}
	secondAdapter := &fakeChatAdapter{chatResp: chatResponse("fallback response")}
	classifier := &fakeRetryClassifier{retryable: false}
	requestLog := newFakeRequestLogService()
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
		requestLog,
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
	if len(requestLog.createAttempts) != 1 {
		t.Fatalf("expected only first attempt to be created, got %d", len(requestLog.createAttempts))
	}
	if len(requestLog.markAttemptFailedArgs) != 1 {
		t.Fatalf("expected first attempt to fail once, got %d", len(requestLog.markAttemptFailedArgs))
	}
	if requestLog.markAttemptFailedArgs[0].ErrorCode != "adapter_error" {
		t.Fatalf("expected adapter_error, got %q", requestLog.markAttemptFailedArgs[0].ErrorCode)
	}
	if len(requestLog.markRequestFailedArgs) != 1 {
		t.Fatalf("expected request to fail once, got %d", len(requestLog.markRequestFailedArgs))
	}
	if requestLog.markRequestFailedArgs[0].ErrorCode != "adapter_error" {
		t.Fatalf("expected adapter_error, got %q", requestLog.markRequestFailedArgs[0].ErrorCode)
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
	service := NewChatCompletionService(router, registry, nil, newFakeRequestLogService())

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
		newFakeRequestLogService(),
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
		newFakeRequestLogService(),
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
		newFakeRequestLogService(),
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
