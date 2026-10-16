package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JorisJonkers-dev/template-go/internal/server"
)

const (
	defaultAddr = ":8080"
	// shutdownGrace is how long in-flight requests get to finish after SIGTERM.
	shutdownGrace = 10 * time.Second
)

// start is main minus os.Exit, so tests can drive it.
func start(parent context.Context, getenv func(string) string) int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr(getenv))
	if err == nil {
		err = server.New(logger, version, shutdownGrace).Serve(ctx, ln)
	}
	if err != nil {
		logger.Error("template-go stopped", "error", err)
		return 1
	}
	return 0
}

// addr is where to listen: ADDR, or the default.
func addr(getenv func(string) string) string {
	if a := getenv("ADDR"); a != "" {
		return a
	}
	return defaultAddr
}
