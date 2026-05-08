package ratelimit

import "testing"

func TestRedisKeyForSubject(t *testing.T) {
	got := redisKeyForSubject("api_key:1")
	want := "unio:ratelimit:api_key:1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
