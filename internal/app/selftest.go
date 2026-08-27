package app

import (
	"context"
	"log"
	"time"

	"github.com/pietjan/bender/internal/bridge"
)

// Selftest arms the -selftest sequence: after startup the app drives its
// own settings flow — open, add, delete, close — and quits. It exists to
// reproduce lifecycle bugs headlessly; each step logs before it runs.
func (a *App) Selftest() { a.selftest = true }

func (a *App) runSelftest(ctx context.Context) {
	steps := []struct {
		name string
		run  func()
	}{
		{"open settings", func() { a.openSettings(ctx) }},
		{"add discord preset", func() {
			if err := a.addService(ctx, bridge.AddService{Preset: "discord"}); err != nil {
				log.Printf("selftest: add: %v", err)
			}
		}},
		{"remove newest service", func() { a.selftestRemoveNewest(ctx) }},
		{"remove newest service again", func() { a.selftestRemoveNewest(ctx) }},
		{"close settings", func() { a.closeSettings() }},
		{"fire web notification", func() {
			if view, ok := a.views[-1]; ok {
				view.PostJSON(`{"type":"fire-notification"}`)
			}
		}},
		{"quit", func() {
			log.Printf("selftest: PASS")
			a.shutdown(ctx)
		}},
	}
	for i, step := range steps {
		step := step
		delay := time.Duration(3+2*i) * time.Second
		time.AfterFunc(delay, func() {
			a.backend.Dispatch(func() {
				log.Printf("selftest: %s", step.name)
				step.run()
			})
		})
	}
}

func (a *App) selftestRemoveNewest(ctx context.Context) {
	rows, err := a.store.ListAllServices(ctx)
	if err != nil || len(rows) == 0 {
		log.Printf("selftest: list: %v", err)
		return
	}
	newest := rows[0].ID
	for _, r := range rows {
		if r.ID > newest {
			newest = r.ID
		}
	}
	if err := a.removeService(ctx, newest); err != nil {
		log.Printf("selftest: remove %d: %v", newest, err)
	}
	a.renderSettings(ctx)
}
