package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/ThankCat/unio-api/internal/failure"
)

const (
	// DefaultMaxJSONBodyBytes 是默认 JSON 请求体最大字节数。
	DefaultMaxJSONBodyBytes int64 = 1 << 20
)

var (
	// ErrRequestBodyTooLarge 表示 JSON 请求体超过允许大小。
	ErrRequestBodyTooLarge = errors.New("request body too large")

	// ErrUnsupportedContentType 表示请求 Content-Type 不是 JSON。
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrEmptyJSONBody 表示请求体为空。
	ErrEmptyJSONBody = errors.New("empty json body")

	// ErrTrailingJSONToken 表示一个 JSON body 后面还有额外 token。
	ErrTrailingJSONToken = errors.New("trailing json token")
)

// DecodeJSON 从 HTTP 请求体读取 JSON，并解码到 dst。
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		return failure.Wrap(
			failure.CodeHTTPUnsupportedContentType,
			ErrUnsupportedContentType,
			failure.WithMessage("content type must be application/json"),
		)
	}

	r.Body = http.MaxBytesReader(w, r.Body, DefaultMaxJSONBodyBytes)

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(dst); err != nil {
		return normalizeJSONDecodeError(err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			return failure.Wrap(
				failure.CodeHTTPRequestBodyTooLarge,
				ErrRequestBodyTooLarge,
				failure.WithMessage("request body too large"),
			)
		}

		return failure.Wrap(
			failure.CodeHTTPTrailingJSONToken,
			ErrTrailingJSONToken,
			failure.WithMessage("request body must contain a single JSON object"),
		)
	}

	return failure.Wrap(
		failure.CodeHTTPTrailingJSONToken,
		ErrTrailingJSONToken,
		failure.WithMessage("request body must contain a single JSON object"),
	)
}

// normalizeJSONDecodeError 将底层 JSON decode 错误收敛成 HTTP 层可稳定识别的错误。
func normalizeJSONDecodeError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return failure.Wrap(
			failure.CodeHTTPRequestBodyTooLarge,
			ErrRequestBodyTooLarge,
			failure.WithMessage("request body too large"),
		)
	}

	if errors.Is(err, io.EOF) {
		return failure.Wrap(
			failure.CodeHTTPEmptyJSONBody,
			ErrEmptyJSONBody,
			failure.WithMessage("request body is required"),
		)
	}

	return failure.Wrap(
		failure.CodeHTTPInvalidJSONBody,
		err,
		failure.WithMessage("invalid json body"),
	)
}

// isJSONContentType 判断 contentType 是否是 "application/json" 类型。
func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return true
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return strings.EqualFold(mediaType, "application/json")
}
