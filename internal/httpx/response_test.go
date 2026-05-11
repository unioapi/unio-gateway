package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteSSEWritesDataEvent(t *testing.T) {
	// 构造支持 Flush 的响应记录器。
	rec := httptest.NewRecorder()

	// 写入一段已经编码好的 JSON payload。
	payload := []byte(`{"param":"value"}`)
	err := WriteSSE(rec, payload)
	if err != nil {
		t.Fatalf("write sse: %v", err)

	}

	// 读取实际写出的 SSE 响应 body。
	gotBody := rec.Body.String()

	// 断言 Content-Type 是 text/event-stream。
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("write sse: got %q, want text/event-stream", rec.Header().Get("Content-Type"))
	}

	// 断言 Cache-Control 是 no-cache。
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("write sse: got %q, want no-cache", rec.Header().Get("Cache-Control"))
	}

	// 断言 body 等于 data: <payload>\n\n。
	wantBody := "data: {\"param\":\"value\"}\n\n"
	if gotBody != wantBody {
		t.Fatalf("write sse body: got %q, want %q", gotBody, wantBody)
	}
	// 断言 recorder 已经 flush。
	if rec.Flushed != true {
		t.Fatalf("write sse: got flushed %v, want true", rec.Flushed)
	}
}

// nonFlusherResponseWriter 是测试用的 ResponseWriter，故意不实现 http.Flusher。
type nonFlusherResponseWriter struct {
	header http.Header
}

// Header 返回测试响应头。
func (n *nonFlusherResponseWriter) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}

	return n.header
}

// Write 返回写入字节数，测试中不需要保存响应体。
func (n *nonFlusherResponseWriter) Write(bytes []byte) (int, error) {
	return len(bytes), nil
}

// WriteHeader 接收状态码，测试中不需要记录。
func (n *nonFlusherResponseWriter) WriteHeader(statusCode int) {}

func TestWriteSSEReturnsErrorWithoutFlusher(t *testing.T) {
	// 构造一个只实现 http.ResponseWriter、不实现 http.Flusher 的 writer。
	writer := nonFlusherResponseWriter{}

	// 调用 WriteSSE，验证不支持 flush 时的错误路径。
	err := WriteSSE(&writer, []byte(`{"param":"value"}`))

	// 断言返回 ErrStreamingUnsupported。
	if !errors.Is(err, ErrStreamingUnsupported) {
		t.Fatalf("write sse: %v", err)
	}
}
