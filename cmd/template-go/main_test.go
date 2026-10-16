package main

import (
	"context"
	"testing"
	"time"
)

func env(addr string) func(string) string {
	return func(key string) string {
		if key == "ADDR" {
			return addr
		}
		return ""
	}
}

func TestAnAddressThatCannotBeListenedOnExitsWithOne(t *testing.T) {
	if code := start(t.Context(), env("not-an-address")); code != 1 {
		t.Fatalf("start = %d, want 1", code)
	}
}

func TestAServiceStoppedByItsContextExitsWithZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	if code := start(ctx, env("127.0.0.1:0")); code != 0 {
		t.Fatalf("start = %d, want 0", code)
	}
}

func TestWithoutAnAddressTheServiceListensOnTheDefault(t *testing.T) {
	if got := addr(env("")); got != defaultAddr {
		t.Fatalf("addr = %q, want %q", got, defaultAddr)
	}
}
