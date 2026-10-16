package server_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/template-go/internal/server"
)

func newServer() *server.Server {
	return server.New(slog.New(slog.DiscardHandler), "test", 10*time.Second)
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func probe(t *testing.T, h http.Handler, path string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec.Code
}

func TestBeforeServingTheServiceIsAliveButNotReady(t *testing.T) {
	h := newServer().Handler()
	if got := probe(t, h, "/healthz"); got != http.StatusOK {
		t.Errorf("GET /healthz = %d, want 200", got)
	}
	if got := probe(t, h, "/readyz"); got != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz = %d, want 503", got)
	}
}

func TestAnUnknownPathIsNotFound(t *testing.T) {
	if got := probe(t, newServer().Handler(), "/nope"); got != http.StatusNotFound {
		t.Fatalf("GET /nope = %d, want 404", got)
	}
}

// serving starts s on a fresh listener and returns its URL, a cancel that stops it, and what Serve
// returned once it has.
func serving(t *testing.T, s *server.Server) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln) }()
	url := "http://" + ln.Addr().String()
	waitFor(t, func() bool { return get(t, url+"/readyz") == http.StatusOK })
	return url, cancel, done
}

func stopped(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("Serve did not return after cancel")
		return nil
	}
}

func TestAServingServiceIsReady(t *testing.T) {
	url, _, _ := serving(t, newServer())
	if got := get(t, url+"/healthz"); got != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", got)
	}
}

func TestCancellingStopsTheServiceCleanly(t *testing.T) {
	_, cancel, done := serving(t, newServer())
	cancel()
	if err := stopped(t, done); err != nil {
		t.Fatalf("Serve = %v, want nil", err)
	}
}

func TestARequestStillOpenWhenTheGraceRunsOutIsAnError(t *testing.T) {
	url, cancel, done := serving(t, server.New(slog.New(slog.DiscardHandler), "test", 50*time.Millisecond))
	var d net.Dialer
	conn, err := d.DialContext(t.Context(), "tcp", url[len("http://"):])
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	// Half a request makes the connection active, and the drain waits for it.
	if _, err := io.WriteString(conn, "GET /healthz HTTP/1.1\r\nHost: x\r\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	cancel()
	if err := stopped(t, done); err == nil {
		t.Fatal("Serve = nil, want the drain's deadline")
	}
}

func TestServingOnAClosedListenerFails(t *testing.T) {
	ln := listen(t)
	_ = ln.Close()
	if err := newServer().Serve(t.Context(), ln); err == nil {
		t.Fatal("Serve on a closed listener returned nil")
	}
}

func get(t *testing.T, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 5s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
