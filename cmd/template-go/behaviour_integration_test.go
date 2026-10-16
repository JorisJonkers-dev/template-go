//go:build integration

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

// freeAddr is a loopback address nothing listens on.
func freeAddr(t *testing.T) string {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().String()
}

func status(t *testing.T, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestTheServiceAnswersItsProbesAndExitsZeroOnSIGTERM(t *testing.T) {
	addr := freeAddr(t)
	exited := make(chan int, 1)
	go func() { exited <- start(context.Background(), env(addr)) }()

	deadline := time.Now().Add(5 * time.Second)
	for status(t, "http://"+addr+"/readyz") != http.StatusOK {
		if time.Now().After(deadline) {
			t.Fatal("the service was not ready within 5s")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := status(t, "http://"+addr+"/healthz"); got != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", got)
	}

	// The service has its SIGTERM handler by now, so the signal stops it, not the test binary.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-exited:
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("the service did not stop on SIGTERM")
	}
}
