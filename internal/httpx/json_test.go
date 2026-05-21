package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThankCat/unio-api/internal/failure"
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
	assertDecodeJSONFailure(t, err, failure.CodeHTTPUnsupportedContentType)
}

func TestDecodeJSONReturnsErrorForEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrEmptyJSONBody) {
		t.Fatalf("expected ErrEmptyJSONBody, got %v", err)
	}
	assertDecodeJSONFailure(t, err, failure.CodeHTTPEmptyJSONBody)
}

func TestDecodeJSONReturnsErrorForInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{`))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	assertDecodeJSONFailure(t, err, failure.CodeHTTPInvalidJSONBody)
}

func TestDecodeJSONReturnsErrorForTrailingJSONToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"value":"hello"} {"value":"second"}`))
	rec := httptest.NewRecorder()

	var body decodeJSONTestBody
	err := DecodeJSON(rec, req, &body)
	if !errors.Is(err, ErrTrailingJSONToken) {
		t.Fatalf("expected ErrTrailingJSONToken, got %v", err)
	}
	assertDecodeJSONFailure(t, err, failure.CodeHTTPTrailingJSONToken)
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
	assertDecodeJSONFailure(t, err, failure.CodeHTTPRequestBodyTooLarge)
}

func assertDecodeJSONFailure(t *testing.T, err error, wantCode failure.Code) {
	t.Helper()

	if failure.CodeOf(err) != wantCode {
		t.Fatalf("expected failure code %q, got %q", wantCode, failure.CodeOf(err))
	}
	if failure.CategoryOf(err) != failure.CategoryHTTP {
		t.Fatalf("expected failure category %q, got %q", failure.CategoryHTTP, failure.CategoryOf(err))
	}
	if fields := failure.FieldsOf(err); len(fields) != 0 {
		t.Fatalf("expected no failure fields, got %#v", fields)
	}
}
