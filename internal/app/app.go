// Package app wires the pieces together: services from the store become
// webviews on a window, title changes become badges, sidebar clicks become
// visibility flips. It speaks only to the platform interfaces and the pure
// packages; nothing here is Windows-specific.
package app

import (
	"context"
	"encoding/json"
	"log"

	"github.com/pietjan/bender/internal/badge"
	"github.com/pietjan/bender/internal/bridge"
	"github.com/pietjan/bender/internal/chrome"
	"github.com/pietjan/bender/internal/platform"
	"github.com/pietjan/bender/internal/service"
	"github.com/pietjan/bender/internal/store"
)

const (
	settingGeometry = "window_geometry"
	settingActive   = "active_service_id"
)

// App is the running application. All methods run on the UI thread.
type App struct {
	backend platform.Backend
	store   *store.Store

	win    platform.Window
	chrome platform.WebView
	views  map[int64]platform.WebView

	services []service.Service
	rules    map[int64]badge.Rule
	badges   map[int64]badge.Badge
	activeID int64

	width, height, dpi int
}

// New assembles the app. Call Run to start it.
func New(backend platform.Backend, st *store.Store) *App {
	return &App{
		backend: backend,
		store:   st,
		views:   map[int64]platform.WebView{},
		rules:   map[int64]badge.Rule{},
		badges:  map[int64]badge.Badge{},
		dpi:     96,
	}
}

// Run builds the window and webviews and enters the event loop. It blocks
// until the app quits.
func (a *App) Run(ctx context.Context) error {
	var err error
	a.services, err = a.store.Services(ctx)
	if err != nil {
		return err
	}
	for _, svc := range a.services {
		a.rules[svc.ID] = a.ruleFor(svc)
	}
	a.activeID = a.restoreActive(ctx)

	a.win, err = a.backend.NewWindow("bender", a.restoreGeometry(ctx))
	if err != nil {
		return err
	}
	a.win.OnResize(func(w, h, dpi int) {
		a.width, a.height, a.dpi = w, h, dpi
		a.applyLayout()
	})
	a.win.OnCloseRequest(func() bool {
		a.win.Hide()
		return false // close means hide; Quit lives in the tray menu
	})

	tray := a.win.Tray()
	tray.SetTooltip("bender")
	tray.SetMenu([]platform.MenuItem{
		{Label: "Show bender", OnClick: a.win.Show},
		{},
		{Label: "Quit", OnClick: func() { a.shutdown(ctx) }},
	})
	tray.OnActivate(a.win.Show)

	a.chrome, err = a.backend.NewWebView(a.win, "")
	if err != nil {
		return err
	}
	a.chrome.OnMessage(func(raw string) { a.onChromeMessage(ctx, raw) })
	a.chrome.NavigateHTML(chrome.Shell())

	for _, svc := range a.services {
		if err := a.addServiceView(svc); err != nil {
			return err
		}
	}

	a.applyLayout()
	a.win.Show()
	return a.backend.Run()
}

func (a *App) addServiceView(svc service.Service) error {
	view, err := a.backend.NewWebView(a.win, svc.Profile)
	if err != nil {
		return err
	}
	id := svc.ID
	view.InitScript(notificationShim)
	view.OnTitleChanged(func(title string) { a.onTitle(id, title) })
	view.OnMessage(func(raw string) { a.onServiceMessage(id, raw) })
	view.SetVisible(id == a.activeID)
	view.Navigate(svc.URL)
	a.views[id] = view
	return nil
}

// applyLayout positions the chrome and every service view. Hidden views
// get the content bounds too, so switching never shows a stale size.
func (a *App) applyLayout() {
	if a.width == 0 || a.height == 0 {
		return
	}
	l := ComputeLayout(a.width, a.height, a.dpi)
	if a.chrome != nil {
		a.chrome.SetBounds(l.Sidebar)
	}
	for _, v := range a.views {
		v.SetBounds(l.Content)
	}
}

func (a *App) onChromeMessage(ctx context.Context, raw string) {
	msg, err := bridge.Decode(raw)
	if err != nil {
		log.Printf("chrome: %v", err)
		return
	}
	switch m := msg.(type) {
	case bridge.Ready:
		a.renderChrome()
	case bridge.Activate:
		a.activate(ctx, m.ServiceID)
	}
}

func (a *App) onServiceMessage(id int64, raw string) {
	msg, err := bridge.Decode(raw)
	if err != nil {
		log.Printf("service %d: %v", id, err)
		return
	}
	if n, ok := msg.(bridge.Notify); ok {
		title := n.Title
		if svc, ok := a.serviceByID(id); ok && title == "" {
			title = svc.Name
		}
		a.win.Tray().Notify(title, n.Body)
	}
}

func (a *App) onTitle(id int64, title string) {
	rule, ok := a.rules[id]
	if !ok {
		return
	}
	next, changed := badge.Parse(rule, a.badges[id], title)
	if !changed || next == a.badges[id] {
		return
	}
	a.badges[id] = next
	a.renderChrome()
}

func (a *App) activate(ctx context.Context, id int64) {
	if _, ok := a.views[id]; !ok || id == a.activeID {
		return
	}
	if prev, ok := a.views[a.activeID]; ok {
		prev.SetVisible(false)
	}
	a.activeID = id
	view := a.views[id]
	view.SetVisible(true)
	view.Focus()
	a.renderChrome()
	if err := a.store.PutSetting(ctx, store.PutSettingParams{Key: settingActive, Value: jsonInt(id)}); err != nil {
		log.Printf("store: save active: %v", err)
	}
}

// renderChrome re-renders the sidebar from current state and pushes it.
func (a *App) renderChrome() {
	if a.chrome == nil {
		return
	}
	html, err := chrome.Render(a.chromeState())
	if err != nil {
		log.Printf("chrome: render: %v", err)
		return
	}
	msg, err := bridge.Encode(bridge.Render{HTML: html})
	if err != nil {
		log.Printf("chrome: encode: %v", err)
		return
	}
	a.chrome.PostJSON(msg)
}

func (a *App) chromeState() chrome.State {
	items := make([]chrome.Item, len(a.services))
	for i, svc := range a.services {
		items[i] = chrome.Item{
			ID:     svc.ID,
			Name:   svc.Name,
			Active: svc.ID == a.activeID,
			Badge:  a.badges[svc.ID],
		}
	}
	return chrome.State{Items: items}
}

func (a *App) ruleFor(svc service.Service) badge.Rule {
	if svc.BadgeRegex != "" {
		rule, err := badge.Compile(svc.BadgeRegex)
		if err == nil {
			return rule
		}
		log.Printf("service %s: bad badge regex %q: %v", svc.Name, svc.BadgeRegex, err)
	}
	return badge.ForPreset(svc.Preset)
}

func (a *App) serviceByID(id int64) (service.Service, bool) {
	for _, svc := range a.services {
		if svc.ID == id {
			return svc, true
		}
	}
	return service.Service{}, false
}

// restoreActive picks the service to show first: the persisted one when it
// still exists, otherwise the first in the list.
func (a *App) restoreActive(ctx context.Context) int64 {
	if len(a.services) == 0 {
		return 0
	}
	if raw, err := a.store.GetSetting(ctx, settingActive); err == nil {
		var id int64
		if json.Unmarshal([]byte(raw), &id) == nil {
			if _, ok := a.serviceByID(id); ok {
				return id
			}
		}
	}
	return a.services[0].ID
}

type geometry struct {
	X, Y, W, H int
}

func (a *App) restoreGeometry(ctx context.Context) platform.Rect {
	fallback := platform.Rect{X: 100, Y: 100, W: 1280, H: 800}
	raw, err := a.store.GetSetting(ctx, settingGeometry)
	if err != nil {
		return fallback
	}
	var g geometry
	if json.Unmarshal([]byte(raw), &g) != nil || g.W <= 0 || g.H <= 0 {
		return fallback
	}
	return platform.Rect{X: g.X, Y: g.Y, W: g.W, H: g.H}
}

// shutdown persists state, releases the webviews so their sessions flush,
// and quits the event loop.
func (a *App) shutdown(ctx context.Context) {
	b := a.win.Bounds()
	if raw, err := json.Marshal(geometry{X: b.X, Y: b.Y, W: b.W, H: b.H}); err == nil {
		if err := a.store.PutSetting(ctx, store.PutSettingParams{Key: settingGeometry, Value: string(raw)}); err != nil {
			log.Printf("store: save geometry: %v", err)
		}
	}
	for _, v := range a.views {
		v.Close()
	}
	a.chrome.Close()
	a.backend.Quit()
}

func jsonInt(v int64) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
