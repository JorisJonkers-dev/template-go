// Command template-go serves HTTP with liveness and readiness probes and drains on SIGTERM.
package main

import (
	"context"
	"os"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

// main is os.Exit around start, which the tests drive; the coverage gate leaves this file out.
func main() {
	os.Exit(start(context.Background(), os.Getenv))
}
