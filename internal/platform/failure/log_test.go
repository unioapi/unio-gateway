package failure

import (
	"errors"
	"testing"
)

func TestLogArgsReturnsNilForNilError(t *testing.T) {
	if args := LogArgs(nil); args != nil {
		t.Fatalf("expected nil args, got %#v", args)
	}
}

func TestLogArgsForPlainError(t *testing.T) {
	err := errors.New("plain error")

	args := LogArgs(err)

	assertLogArg(t, args, "error", err)
	assertNoLogArg(t, args, "error_code")
	assertNoLogArg(t, args, "error_category")
}

func TestLogArgsForFailure(t *testing.T) {
	err := New(
		CodeConfigInvalid,
		WithMessage("parse REDIS_DB as int"),
	)

	args := LogArgs(err)

	assertLogArg(t, args, "error", err)
	assertLogArg(t, args, "error_code", string(CodeConfigInvalid))
	assertLogArg(t, args, "error_category", string(CategoryConfig))
}

func TestLogArgsIncludesFailureFields(t *testing.T) {
	err := New(
		CodeConfigInvalid,
		WithField("config_key", "REDIS_DB"),
		WithField("", "ignored"),
	)

	args := LogArgs(err)

	assertLogArg(t, args, "config_key", "REDIS_DB")
	assertNoLogArg(t, args, "")
}

func assertLogArg(t *testing.T, args []any, key string, want any) {
	t.Helper()

	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == key {
			if args[i+1] != want {
				t.Fatalf("expected log arg %s=%#v, got %#v in %#v", key, want, args[i+1], args)
			}
			return
		}
	}

	t.Fatalf("expected log arg %s=%#v in %#v", key, want, args)
}

func assertNoLogArg(t *testing.T, args []any, key string) {
	t.Helper()

	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == key {
			t.Fatalf("expected no log arg %s, got %#v", key, args)
		}
	}
}
