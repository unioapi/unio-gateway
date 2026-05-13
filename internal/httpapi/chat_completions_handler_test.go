package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpx"
)

// fakeAPIKeyAuthenticator 是 chat completions 测试使用的 API Key 认证器。
type fakeAPIKeyAuthenticator struct {
	principal *auth.APIKeyPrincipal
	err       error
	token     string
}

// AuthenticateAPIKey 记录收到的 token，并返回测试预设的认证结果。
func (a *fakeAPIKeyAuthenticator) AuthenticateAPIKey(ctx context.Context, plaintext string) (*auth.APIKeyPrincipal, error) {
	a.token = plaintext
	return a.principal, a.err
}

func TestRouterV1ChatCompletionWithMissingAPIKey(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	handler := newTestRouter(authenticator, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRouterV1ChatCompletionWithAPIKey(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	handler := newTestRouter(authenticator, nil, nil)

	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body ChatCompletionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Object != "chat.completion" {
		t.Fatalf("expected object %q, got %q", "chat.completion", body.Object)
	}

	if body.Model != reqBody.Model {
		t.Fatalf("expected model %q, got %q", reqBody.Model, body.Model)
	}

	if len(body.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(body.Choices))
	}
}

func TestRouterV1ChatCompletionWithInvalidBody(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	handler := newTestRouter(authenticator, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", body.Error.Code)
	}

	if body.Error.Message != "invalid json body" {
		t.Fatalf("expected message %q, got %q", "invalid json body", body.Error.Message)
	}
}

func TestRouterV1ChatCompletionWithMissingModel(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	handler := newTestRouter(authenticator, nil, nil)

	reqBody := ChatCompletionRequest{}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", body.Error.Code)
	}

	if body.Error.Message != "model is required" {
		t.Fatalf("expected message %q, got %q", "model is required", body.Error.Message)
	}
}

func TestRouterV1ChatCompletionWithMissingMessages(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	handler := newTestRouter(authenticator, nil, nil)

	reqBody := ChatCompletionRequest{
		Model:    "openai/gpt-4.1",
		Messages: []ChatMessage{},
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", body.Error.Code)
	}

	if body.Error.Message != "messages is required" {
		t.Fatalf("expected message %q, got %q", "messages is required", body.Error.Message)
	}
}

// fakeChatCompletionService 是 chat completions 测试使用的 service 替身。
type fakeChatCompletionService struct {
	createCalled       bool
	streamCalled       bool
	req                ChatCompletionRequest
	createResp         *ChatCompletionResponse
	streamResp         []ChatCompletionStreamResponse
	err                error
	streamErrAfterEmit error
}

// CreateChatCompletion 记录 handler 传入的请求，并返回测试预设的响应。
func (s *fakeChatCompletionService) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	s.createCalled = true
	s.req = req
	return s.createResp, s.err
}

// StreamChatCompletion 记录 handler 传入的流式请求，并逐个发出测试预设响应。
func (s *fakeChatCompletionService) StreamChatCompletion(ctx context.Context, req ChatCompletionRequest, emit func(ChatCompletionStreamResponse) error) error {
	s.streamCalled = true
	s.req = req

	if s.err != nil {
		return s.err
	}

	for _, chunk := range s.streamResp {
		if err := emit(chunk); err != nil {
			return err
		}
	}

	return s.streamErrAfterEmit
}

func TestRouterV1ChatCompletionCallsService(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}

	service := &fakeChatCompletionService{
		createResp: &ChatCompletionResponse{
			Object: "chat.completion",
			Model:  "openai/gpt-4.1",
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "mock response",
					},
					FinishReason: "stop",
				},
			},
		},
	}

	handler := newTestRouter(authenticator, service, nil)

	buf := new(bytes.Buffer)
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !service.createCalled {
		t.Fatal("expected chat completion service to be called")
	}

	if service.req.Model != reqBody.Model {
		t.Fatalf("expected service model %q, got %q", reqBody.Model, service.req.Model)
	}

	if len(service.req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(service.req.Messages))
	}

	var recBody ChatCompletionResponse
	if err := json.NewDecoder(rec.Body).Decode(&recBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if recBody.Model != service.createResp.Model {
		t.Fatalf("expected response model %q, got %q", service.createResp.Model, recBody.Model)
	}
}

func TestRouterV1ChatCompletionPreservesExplicitZeroTemperature(t *testing.T) {
	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	service := &fakeChatCompletionService{
		createResp: &ChatCompletionResponse{
			Object: "chat.completion",
			Model:  "openai/gpt-4.1",
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "mock response",
					},
					FinishReason: "stop",
				},
			},
		},
	}
	handler := newTestRouter(authenticator, service, nil)

	zero := 0.0
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: &zero,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !service.createCalled {
		t.Fatal("expected chat completion service to be called")
	}

	if service.req.Temperature == nil {
		t.Fatal("expected temperature to be preserved")
	}

	if *service.req.Temperature != 0 {
		t.Fatalf("expected temperature 0, got %v", *service.req.Temperature)
	}
}

func TestRouterV1ChatCompletionWithInvalidTemperature(t *testing.T) {
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: float64Ptr(2.1),
	}

	assertChatCompletionInvalidRequest(t, reqBody, "temperature must be between 0 and 2", "temperature")
}

func TestRouterV1ChatCompletionWithInvalidTopP(t *testing.T) {
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		TopP: float64Ptr(1.1),
	}

	assertChatCompletionInvalidRequest(t, reqBody, "top_p must be between 0 and 1", "top_p")
}

func TestRouterV1ChatCompletionWithInvalidMaxTokens(t *testing.T) {
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: intPtr(0),
	}

	assertChatCompletionInvalidRequest(t, reqBody, "max_tokens must be greater than 0", "max_tokens")
}

func assertChatCompletionInvalidRequest(t *testing.T, reqBody ChatCompletionRequest, wantMessage string, wantParam string) {
	t.Helper()

	authenticator := &fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}
	service := &fakeChatCompletionService{}
	handler := newTestRouter(authenticator, service, nil)

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if service.createCalled {
		t.Fatal("expected chat completion service not to be called")
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", body.Error.Code)
	}

	if body.Error.Message != wantMessage {
		t.Fatalf("expected message %q, got %q", wantMessage, body.Error.Message)
	}

	if body.Error.Type != "invalid_request_error" {
		t.Fatalf("expected error type %q, got %q", "invalid_request_error", body.Error.Type)
	}

	if body.Error.Param == nil {
		t.Fatal("expected error param to be set")
	}

	if *body.Error.Param != wantParam {
		t.Fatalf("expected error param %q, got %q", wantParam, *body.Error.Param)
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func TestChatCompletionMissingModelReturnsOpenAIError(t *testing.T) {
	// 构造带认证的 router。
	authenticator := &fakeAPIKeyAuthenticator{principal: &auth.APIKeyPrincipal{
		APIKeyID:  1,
		ProjectID: 1,
		KeyPrefix: "unio_sk_test",
	}}
	router := newTestRouter(authenticator, nil, nil)

	// 发送缺少 model 的请求。
	reqBody := ChatCompletionRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 解析错误响应。
	var recBody httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&recBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	// 断言 HTTP status 是 400。
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	// 断言 error.code 是 invalid_request。
	if recBody.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", recBody.Error.Code)
	}

	// 断言 error.type 是 invalid_request_error。
	if recBody.Error.Type != "invalid_request_error" {
		t.Fatalf("expected error type %q, got %q", "invalid_request_error", recBody.Error.Type)
	}

	// 断言 error.param 是 model。
	if recBody.Error.Param == nil {
		t.Fatal("expected error param to be set")
	}

	if *recBody.Error.Param != "model" {
		t.Fatalf("expected parameter %q, got %q", "model", *recBody.Error.Param)
	}
}

func TestChatCompletionMissingMessagesReturnsOpenAIError(t *testing.T) {
	// 构造带认证的 router。
	authenticator := &fakeAPIKeyAuthenticator{principal: &auth.APIKeyPrincipal{
		APIKeyID:  1,
		ProjectID: 1,
		KeyPrefix: "unio_sk_test",
	}}
	router := newTestRouter(authenticator, nil, nil)

	// 发送缺少 messages 的请求。
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", buf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 解析错误响应。
	var recBody httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&recBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	// 断言 HTTP status 是 400。
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	// 断言 error.code 是 invalid_request。
	if recBody.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", recBody.Error.Code)
	}

	// 断言 error.type 是 invalid_request_error。
	if recBody.Error.Type != "invalid_request_error" {
		t.Fatalf("expected error type %q, got %q", "invalid_request_error", recBody.Error.Type)
	}

	// 断言 error.param 是 messages。
	if recBody.Error.Param == nil {
		t.Fatal("expected error param to be set")
	}

	if *recBody.Error.Param != "messages" {
		t.Fatalf("expected parameter %q, got %q", "messages", *recBody.Error.Param)
	}
}

func TestRouterV1ChatCompletionWithStreamTrueWritesSSE(t *testing.T) {
	service := &fakeChatCompletionService{
		streamResp: []ChatCompletionStreamResponse{
			{
				ID:      "chatcmpl_stream_test",
				Object:  "chat.completion.chunk",
				Created: 123,
				Model:   "openai/gpt-4.1",
				Choices: []ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: ChatCompletionStreamDelta{
							Role:    "assistant",
							Content: "mock response",
						},
						FinishReason: nil,
					},
				},
			},
		},
	}

	// 构造带认证的 router。
	router := newTestRouter(&fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}, service, nil)

	// 发送 stream=true 的 chat completions 请求。
	stream := true
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: &stream,
	}
	reqBuf := new(bytes.Buffer)
	if err := json.NewEncoder(reqBuf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", reqBuf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 断言 HTTP status 是 200。
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// 断言 Content-Type 是 text/event-stream。
	if rec.Header().Get("Content-Type") != httpx.ContentTypeSSE {
		t.Fatalf("expected Content-Type %q, got %q", "text/event-stream", rec.Header().Get("Content-Type"))
	}

	// 读取响应 body。
	gotBody := rec.Body.String()

	// 断言 body 包含 data: 前缀。
	if !strings.Contains(gotBody, "data: ") {
		t.Fatalf("expected body %q, got %q", "data:", gotBody)
	}

	// 断言 body 包含 chat.completion.chunk。
	if !strings.Contains(gotBody, "chat.completion.chunk") {
		t.Fatalf("expected body %q, got %q", "chat.completion.chunk", gotBody)
	}

	// 断言 body 包含 mock response。
	if !strings.Contains(gotBody, "mock response") {
		t.Fatalf("expected body %q, got %q", "mock response", gotBody)
	}

	// 断言 body 包含 data: [DONE]。
	if !strings.Contains(gotBody, "data: [DONE]") {
		t.Fatalf("expected body to contain %q, got %q", "data: [DONE]", gotBody)
	}

	if !service.streamCalled {
		t.Fatal("expected stream service to be called")
	}

	if service.createCalled {
		t.Fatal("expected create service not to be called")
	}

	if service.req.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", service.req.Model)
	}

	if service.req.Stream == nil || !*service.req.Stream {
		t.Fatal("expected stream to be true")
	}

	if len(service.req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(service.req.Messages))
	}

	if service.req.Messages[0].Role != "user" {
		t.Fatalf("expected message role %q, got %q", "user", service.req.Messages[0].Role)
	}

	if service.req.Messages[0].Content != "Hello" {
		t.Fatalf("expected message content %q, got %q", "Hello", service.req.Messages[0].Content)
	}
}

func TestRouterV1ChatCompletionStreamReturnsJSONErrorBeforeFirstChunk(t *testing.T) {
	service := &fakeChatCompletionService{
		err: context.DeadlineExceeded,
	}
	router := newTestRouter(&fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}, service, nil)

	stream := true
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: &stream,
	}
	reqBuf := new(bytes.Buffer)
	if err := json.NewEncoder(reqBuf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", reqBuf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	gotBody := rec.Body.String()

	var body httpx.ErrorResponse
	if err := json.NewDecoder(strings.NewReader(gotBody)).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Error.Code != "stream_chat_completion_error" {
		t.Fatalf("expected error code %q, got %q", "stream_chat_completion_error", body.Error.Code)
	}

	if strings.Contains(gotBody, "data:") {
		t.Fatalf("expected body not to contain %q, got %q", "data:", gotBody)
	}

	if !service.streamCalled {
		t.Fatal("expected stream service to be called")
	}

	if service.createCalled {
		t.Fatal("expected create service not to be called")
	}
}

func TestRouterV1ChatCompletionStreamDoesNotWriteJSONErrorAfterChunkStarted(t *testing.T) {
	service := &fakeChatCompletionService{
		streamResp: []ChatCompletionStreamResponse{
			{
				ID:      "chatcmpl_stream_test",
				Object:  "chat.completion.chunk",
				Created: 123,
				Model:   "openai/gpt-4.1",
				Choices: []ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: ChatCompletionStreamDelta{
							Role:    "assistant",
							Content: "mock response",
						},
						FinishReason: nil,
					},
				},
			},
		},
		streamErrAfterEmit: context.Canceled,
	}
	router := newTestRouter(&fakeAPIKeyAuthenticator{
		principal: &auth.APIKeyPrincipal{
			APIKeyID:  1,
			ProjectID: 1,
			KeyPrefix: "unio_sk_test",
		},
	}, service, nil)

	stream := true
	reqBody := ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: &stream,
	}
	reqBuf := new(bytes.Buffer)
	if err := json.NewEncoder(reqBuf).Encode(reqBody); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", reqBuf)
	req.Header.Set("Authorization", "Bearer unio_sk_test")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != httpx.ContentTypeSSE {
		t.Fatalf("expected Content-Type %q, got %q", httpx.ContentTypeSSE, rec.Header().Get("Content-Type"))
	}

	gotBody := rec.Body.String()
	if !strings.Contains(gotBody, "mock response") {
		t.Fatalf("expected body to contain %q, got %q", "mock response", gotBody)
	}

	if strings.Contains(gotBody, "stream_chat_completion_error") {
		t.Fatalf("expected body not to contain %q, got %q", "stream_chat_completion_error", gotBody)
	}

	if strings.Contains(gotBody, "data: [DONE]") {
		t.Fatalf("expected body not to contain %q, got %q", "data: [DONE]", gotBody)
	}

	if !service.streamCalled {
		t.Fatal("expected stream service to be called")
	}

	if service.createCalled {
		t.Fatal("expected create service not to be called")
	}
}
