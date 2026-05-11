package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeJSONTestBody 是 DecodeJSON 测试使用的请求体结构。
type decodeJSONTestBody struct {
	Value string `json:"value"`
}

func TestDecodeJSONDecodesBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"value":"hello"}`))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	if err := DecodeJSON(rec, req, &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if body.Value != "hello" {
		t.Fatalf("expected value %q, got %q", "hello", body.Value)
	}
}

func TestDecodeJSONReturnsErrorForInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{`))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestDecodeJSONReturnsErrorForTooLargeBody(t *testing.T) {
	largeValue := strings.Repeat("a", int(DefaultMaxJSONBodyBytes)+1)
	reqBody := `{"value":"` + largeValue + `"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrRequestBodyTooLarge) {
		t.Fatalf("expected ErrRequestBodyTooLarge, got %v", err)
	}
}
