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

func TestDecodeJSONAcceptsJSONContentTypeWithCharset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"value":"hello"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	if err := DecodeJSON(rec, req, &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if body.Value != "hello" {
		t.Fatalf("expected value %q, got %q", "hello", body.Value)
	}
}

func TestDecodeJSONReturnsErrorForUnsupportedContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"value":"hello"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrUnsupportedContentType) {
		t.Fatalf("expected ErrUnsupportedContentType, got %v", err)
	}
}

func TestDecodeJSONReturnsErrorForEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrEmptyJSONBody) {
		t.Fatalf("expected ErrEmptyJSONBody, got %v", err)
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

func TestDecodeJSONReturnsErrorForTrailingJSONToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"value":"hello"} {"value":"second"}`))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrTrailingJSONToken) {
		t.Fatalf("expected ErrTrailingJSONToken, got %v", err)
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
