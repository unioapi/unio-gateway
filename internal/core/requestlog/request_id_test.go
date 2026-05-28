package requestlog

import (
	"strings"
	"testing"
)

func TestGenerateRequestIDReturnsServerRequestID(t *testing.T) {
	requestID, err := GenerateRequestID()
	if err != nil {
		t.Fatalf("generate request id: %v", err)
	}

	if !strings.HasPrefix(requestID, "req_") {
		t.Fatalf("expected request id prefix %q, got %q", "req_", requestID)
	}
	if len(requestID) != len("req_")+requestIDRandomBytes*2 {
		t.Fatalf("expected request id length %d, got %d", len("req_")+requestIDRandomBytes*2, len(requestID))
	}
}

func TestGenerateRequestIDGeneratesDifferentValues(t *testing.T) {
	first, err := GenerateRequestID()
	if err != nil {
		t.Fatalf("generate first request id: %v", err)
	}

	second, err := GenerateRequestID()
	if err != nil {
		t.Fatalf("generate second request id: %v", err)
	}

	if first == second {
		t.Fatalf("expected different request ids, got %q", first)
	}
}
