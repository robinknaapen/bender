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

	"github.com/pietjan/spectacle"

	"github.com/pietjan/bender/internal/app"
	"github.com/pietjan/bender/internal/appicon"
	"github.com/pietjan/bender/internal/store"
)

func main() {
	// The entire UI — native toolkit, webviews, event loop — lives on
	// this one thread.
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

	// Browser profiles and logs are per-machine data, not roaming config.
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(cache, "bender")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	// Windows -H=windowsgui builds have no console; keep a log file.
	// Console builds (make build/debug) additionally keep stderr.
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

	// The per-OS names predate the spectacle split; existing profiles
	// (logged-in sessions) live under them, so they stay.
	sub := "webkit"
	if runtime.GOOS == "windows" {
		sub = "webview2"
	}
	// Selftests add/remove services (deleting profiles with them); keep
	// their browser data away from the real profiles too.
	if *selftest {
		sub = "selftest-" + sub
	}
	backend, err := spectacle.New(spectacle.Config{
		ID:         "bender",
		Name:       "Bender",
		Comment:    "Multi-service messaging browser",
		Categories: "Network;InstantMessaging;",
		Icon:       appicon.PNG,
		DataDir:    filepath.Join(dataDir, sub),
		Debug:      *debug,
	})
	if err != nil {
		return err
	}
	a := app.New(backend, st, *debug)
	if *selftest {
		a.Selftest()
	}
	return a.Run(ctx)
}
