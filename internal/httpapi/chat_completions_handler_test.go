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
	called bool
	req    ChatCompletionRequest
	resp   *ChatCompletionResponse
	err    error
}

// CreateChatCompletion 记录 handler 传入的请求，并返回测试预设的响应。
func (s *fakeChatCompletionService) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	s.called = true
	s.req = req
	return s.resp, s.err
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
		resp: &ChatCompletionResponse{
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

	if !service.called {
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

	if recBody.Model != service.resp.Model {
		t.Fatalf("expected response model %q, got %q", service.resp.Model, recBody.Model)
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
		resp: &ChatCompletionResponse{
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

	if !service.called {
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

	if service.called {
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
	// TODO: 第1步，构造带认证的 router
	authenticator := &fakeAPIKeyAuthenticator{principal: &auth.APIKeyPrincipal{
		APIKeyID:  1,
		ProjectID: 1,
		KeyPrefix: "unio_sk_test",
	}}
	router := newTestRouter(authenticator, nil, nil)

	// TODO: 第2步，发送缺少 model 的请求
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

	// TODO: 第3步，解析错误响应
	var recBody httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&recBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	// TODO: 断言 HTTP status 是 400
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	// TODO: 断言 error.code 是 invalid_request
	if recBody.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", recBody.Error.Code)
	}

	// TODO: 断言 error.type 是 invalid_request_error
	if recBody.Error.Type != "invalid_request_error" {
		t.Fatalf("expected error type %q, got %q", "invalid_request_error", recBody.Error.Type)
	}

	// TODO: 断言 error.param 是 model
	if recBody.Error.Param == nil {
		t.Fatal("expected error param to be set")
	}

	if *recBody.Error.Param != "model" {
		t.Fatalf("expected parameter %q, got %q", "model", *recBody.Error.Param)
	}
}

func TestChatCompletionMissingMessagesReturnsOpenAIError(t *testing.T) {
	// TODO: 第1步，构造带认证的 router
	authenticator := &fakeAPIKeyAuthenticator{principal: &auth.APIKeyPrincipal{
		APIKeyID:  1,
		ProjectID: 1,
		KeyPrefix: "unio_sk_test",
	}}
	router := newTestRouter(authenticator, nil, nil)

	// TODO: 第2步，发送缺少 messages 的请求
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

	// TODO: 第3步，解析错误响应
	var recBody httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&recBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	// TODO: 断言 HTTP status 是 400
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	// TODO: 断言 error.code 是 invalid_request
	if recBody.Error.Code != "invalid_request" {
		t.Fatalf("expected error code %q, got %q", "invalid_request", recBody.Error.Code)
	}

	// TODO: 断言 error.type 是 invalid_request_error
	if recBody.Error.Type != "invalid_request_error" {
		t.Fatalf("expected error type %q, got %q", "invalid_request_error", recBody.Error.Type)
	}

	// TODO: 断言 error.param 是 messages
	if recBody.Error.Param == nil {
		t.Fatal("expected error param to be set")
	}

	if *recBody.Error.Param != "messages" {
		t.Fatalf("expected parameter %q, got %q", "messages", *recBody.Error.Param)
	}
}
