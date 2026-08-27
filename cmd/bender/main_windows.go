//go:build windows

// Command bender is a multi-service messaging browser: each service runs
// in its own isolated OS webview inside one window.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pietjan/bender/internal/app"
	"github.com/pietjan/bender/internal/platform/win"
	"github.com/pietjan/bender/internal/store"
)

func main() {
	// The entire UI — Win32, COM, WebView2 — lives on this one thread.
	runtime.LockOSThread()
	if err := run(); err != nil {
		log.Printf("bender: %v", err)
		os.Exit(1)
	}
}

func run() error {
	debug := flag.Bool("debug", false, "enable DevTools in webviews")
	dbPath := flag.String("db", "", "database path (default: user config dir)")
	flag.Parse()

	// Browser profiles and logs are per-machine data, not roaming config.
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(cache, "bender")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	// -H=windowsgui builds have no console; keep a log file. Console
	// builds (make build/debug) additionally keep stderr.
	if f, err := os.OpenFile(filepath.Join(dataDir, "bender.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}

	ctx := context.Background()
	path := *dbPath
	if path == "" {
		if path, err = store.DefaultPath(); err != nil {
			return err
		}
	}
	st, err := store.Open(ctx, path)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()
	if err := st.Seed(ctx); err != nil {
		return fmt.Errorf("seed store: %w", err)
	}

	backend, err := win.New(filepath.Join(dataDir, "webview2"), *debug)
	if err != nil {
		return err
	}
	return app.New(backend, st, *debug).Run(ctx)
}
