package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
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
		return ErrUnsupportedContentType
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
			return ErrRequestBodyTooLarge
		}

		return ErrTrailingJSONToken
	}

	return ErrTrailingJSONToken
}

// normalizeJSONDecodeError 将底层 JSON decode 错误收敛成 HTTP 层可稳定识别的错误。
func normalizeJSONDecodeError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return ErrRequestBodyTooLarge
	}

	if errors.Is(err, io.EOF) {
		return ErrEmptyJSONBody
	}

	return err
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
