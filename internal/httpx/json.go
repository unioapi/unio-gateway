package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	// DefaultMaxJSONBodyBytes 是默认 JSON 请求体最大字节数。
	DefaultMaxJSONBodyBytes int64 = 1 << 20
)

var (
	// ErrRequestBodyTooLarge 表示 JSON 请求体超过允许大小。
	ErrRequestBodyTooLarge = errors.New("request body too large")
)

// DecodeJSON 从 HTTP 请求体读取 JSON，并解码到 dst。
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, DefaultMaxJSONBodyBytes)

	decoder := json.NewDecoder(r.Body)

	// TODO(阶段4/production): [GAP-4-002] 当前 JSON 解码未校验 Content-Type 和尾随 JSON token，会让公网 API 接受模糊请求体；开放 OpenAI-compatible API 前；补齐严格 body 校验并把 body too large / malformed JSON 映射为稳定的 OpenAI-compatible error。
	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return ErrRequestBodyTooLarge
		}

		return err
	}

	return nil
}
