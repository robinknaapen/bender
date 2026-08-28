//go:build linux

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
	"github.com/pietjan/bender/internal/platform/linux"
	"github.com/pietjan/bender/internal/store"
)

func main() {
	// The entire UI — GTK, WebKit, GLib main loop — lives on this thread.
	runtime.LockOSThread()
	if err := run(); err != nil {
		log.Printf("bender: %v", err)
		os.Exit(1)
	}
}

func run() error {
	debug := flag.Bool("debug", false, "enable DevTools in webviews")
	selftest := flag.Bool("selftest", false, "drive the settings lifecycle and exit")
	dbPath := flag.String("db", "", "database path (default: user config dir)")
	flag.Parse()

	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(cache, "bender")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if f, err := os.OpenFile(filepath.Join(dataDir, "bender.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}

	ctx := context.Background()
	path := *dbPath
	if path == "" && *selftest {
		// Selftests mutate services; never point them at the real config.
		path = filepath.Join(dataDir, "selftest.db")
		os.Remove(path)
	}
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

	// Selftests add/remove services (deleting profiles with them); keep
	// their browser data away from the real profiles too.
	webviewData := filepath.Join(dataDir, "webkit")
	if *selftest {
		webviewData = filepath.Join(dataDir, "selftest-webkit")
	}
	backend, err := linux.New(webviewData, *debug)
	if err != nil {
		return err
	}
	a := app.New(backend, st, *debug)
	if *selftest {
		a.Selftest()
	}
	return a.Run(ctx)
}
