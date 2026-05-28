package sse

import (
	"errors"
	"strings"
	"testing"
)

func TestReaderParsesFieldsAndMultiLineData(t *testing.T) {
	input := ": keepalive\r\n" +
		"id: evt-1\r\n" +
		"event: completion\r\n" +
		"retry: 1500\r\n" +
		"data: hello\r\n" +
		"data: world\r\n" +
		"\r\n"

	events, err := readAllEvents(input, Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	event := events[0]
	if event.Type != "completion" {
		t.Fatalf("got type %q, want completion", event.Type)
	}
	if string(event.Data) != "hello\nworld" {
		t.Fatalf("got data %q, want multi-line data", event.Data)
	}
	if event.ID == nil || *event.ID != "evt-1" {
		t.Fatalf("got id %+v, want evt-1", event.ID)
	}
	if event.RetryMilliseconds == nil || *event.RetryMilliseconds != 1500 {
		t.Fatalf("got retry %+v, want 1500", event.RetryMilliseconds)
	}
}

func TestReaderSupportsLoneCRLineEndings(t *testing.T) {
	events, err := readAllEvents("data: hello\r\r", Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if string(events[0].Data) != "hello" {
		t.Fatalf("got data %q, want hello", events[0].Data)
	}
}

func TestReaderIgnoresUTF8BOMOnFirstLine(t *testing.T) {
	events, err := readAllEvents("\xEF\xBB\xBFdata: hello\n\n", Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if string(events[0].Data) != "hello" {
		t.Fatalf("got data %q, want hello", events[0].Data)
	}
}

func TestReaderSkipsEventsWithoutData(t *testing.T) {
	input := ": keepalive\n\n" +
		"event: only-type\n\n" +
		"data: visible\n\n"

	events, err := readAllEvents(input, Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if string(events[0].Data) != "visible" {
		t.Fatalf("got data %q, want visible", events[0].Data)
	}
}

func TestReaderDispatchesEmptyDataField(t *testing.T) {
	events, err := readAllEvents("data\n\n", Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if string(events[0].Data) != "" {
		t.Fatalf("got data %q, want empty", events[0].Data)
	}
}

func TestReaderIgnoresInvalidRetryAndNullID(t *testing.T) {
	events, err := readAllEvents("id: ok\nid: bad\x00id\nretry: later\ndata: body\n\n", Config{})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].ID == nil || *events[0].ID != "ok" {
		t.Fatalf("got id %+v, want ok", events[0].ID)
	}
	if events[0].RetryMilliseconds != nil {
		t.Fatalf("got retry %+v, want nil", events[0].RetryMilliseconds)
	}
}

func TestReaderReturnsLineTooLong(t *testing.T) {
	_, err := readAllEvents("data: too long\n\n", Config{
		MaxLineBytes:  4,
		MaxEventBytes: 64,
	})
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("got error %v, want ErrLineTooLong", err)
	}
}

func TestReaderReturnsEventTooLarge(t *testing.T) {
	_, err := readAllEvents("data: abcd\n\n", Config{
		MaxLineBytes:  64,
		MaxEventBytes: 3,
	})
	if !errors.Is(err, ErrEventTooLarge) {
		t.Fatalf("got error %v, want ErrEventTooLarge", err)
	}
}

func TestReaderReturnsMalformedStreamForPendingEventAtEOF(t *testing.T) {
	_, err := readAllEvents("data: unfinished", Config{})
	if !errors.Is(err, ErrMalformedStream) {
		t.Fatalf("got error %v, want ErrMalformedStream", err)
	}
}

func readAllEvents(input string, cfg Config) ([]Event, error) {
	reader := NewReader(strings.NewReader(input), cfg)
	events := make([]Event, 0)
	for reader.Next() {
		events = append(events, reader.Event())
	}

	return events, reader.Err()
}
