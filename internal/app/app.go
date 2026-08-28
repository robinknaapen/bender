// Package app wires the pieces together: services from the store become
// webviews on a window, title changes become badges, sidebar clicks become
// visibility flips. It speaks only to the platform interfaces and the pure
// packages; nothing here is Windows-specific.
package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
	"log"
	"strings"

	"github.com/pietjan/bender/internal/badge"
	"github.com/pietjan/bender/internal/bridge"
	"github.com/pietjan/bender/internal/chrome"
	"github.com/pietjan/bender/internal/platform"
	"github.com/pietjan/bender/internal/service"
	"github.com/pietjan/bender/internal/store"
)

const (
	settingGeometry  = "window_geometry"
	settingActive    = "active_service_id"
	settingCollapsed = "sidebar_collapsed"
)

// App is the running application. All methods run on the UI thread.
type App struct {
	backend  platform.Backend
	store    *store.Store
	debug    bool
	selftest bool

	win      platform.Window
	chrome   platform.WebView
	settings platform.WebView // lazily created on first open
	views    map[int64]platform.WebView

	settingsOpen bool
	settingsErr  string
	collapsed    bool

	services []service.Service
	rules    map[int64]badge.Rule
	badges   map[int64]badge.Badge
	// shimIcon marks services whose icon came from the in-page resolver,
	// which outranks the coarse WebView2 favicon event.
	shimIcon map[int64]bool
	activeID int64
	// lastNotifyID is the service behind the pending notification. Only
	// one balloon can pend at a time (a new one replaces it), so
	// last-wins matches what the user actually clicks.
	lastNotifyID int64

	width, height, dpi int
}

// New assembles the app. Call Run to start it. debug adds the built-in
// Test service for exercising notifications and badges.
func New(backend platform.Backend, st *store.Store, debug bool) *App {
	return &App{
		backend:  backend,
		store:    st,
		debug:    debug,
		views:    map[int64]platform.WebView{},
		rules:    map[int64]badge.Rule{},
		badges:   map[int64]badge.Badge{},
		shimIcon: map[int64]bool{},
		dpi:      96,
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
	a.services = a.withTestService(a.services)
	for _, svc := range a.services {
		a.rules[svc.ID] = a.ruleFor(svc)
	}
	a.activeID = a.restoreActive(ctx)
	if v, err := a.store.GetSetting(ctx, settingCollapsed); err == nil {
		a.collapsed = v == "true"
	}

	a.win, err = a.backend.NewWindow("Bender", a.restoreGeometry(ctx))
	if err != nil {
		return err
	}
	a.win.OnResize(func(w, h, dpi int) {
		a.width, a.height, a.dpi = w, h, dpi
		a.applyLayout()
	})
	tray := a.win.Tray()
	a.win.OnCloseRequest(func() bool {
		// Close means hide; Quit lives in the tray menu. But hiding with
		// no reachable tray icon would strand the app — quit instead
		// (shutdown persists state and ends the loop cleanly).
		if !tray.Available() {
			a.shutdown(ctx)
			return false
		}
		a.win.Hide()
		return false
	})
	tray.SetTooltip("Bender")
	tray.SetMenu([]platform.MenuItem{
		{Label: "Show Bender", OnClick: a.win.Show},
		{},
		{Label: "Quit", OnClick: func() { a.shutdown(ctx) }},
	})
	tray.OnActivate(a.win.Show)
	tray.OnNotificationClick(func() {
		a.win.Show()
		if a.lastNotifyID != 0 {
			a.activate(ctx, a.lastNotifyID)
		}
	})

	a.chrome, err = a.backend.NewWebView(a.win, "")
	if err != nil {
		return err
	}
	// Bridge messages arrive inside WebView2 event handlers, where
	// creating webviews (or anything that pumps messages) deadlocks —
	// WebView2 will not deliver callbacks re-entrantly. Dispatch defers
	// every handler onto a clean stack.
	a.chrome.OnMessage(func(raw string) {
		a.backend.Dispatch(func() { a.onChromeMessage(ctx, raw) })
	})
	a.chrome.NavigateHTML(chrome.Shell())

	a.settings, err = a.backend.NewWebView(a.win, "")
	if err != nil {
		return err
	}
	a.settings.OnMessage(func(raw string) {
		a.backend.Dispatch(func() { a.onSettingsMessage(ctx, raw) })
	})
	a.settings.SetVisible(false)
	a.settings.NavigateHTML(chrome.Shell())

	for _, svc := range a.services {
		if err := a.addServiceView(svc); err != nil {
			return err
		}
	}

	a.applyLayout()
	a.win.Show()
	if a.selftest {
		a.runSelftest(ctx)
	}
	return a.backend.Run()
}

func (a *App) addServiceView(svc service.Service) error {
	view, err := a.backend.NewWebView(a.win, svc.Profile)
	if err != nil {
		return err
	}
	id := svc.ID
	// Two complementary notification paths: the shim intercepts page
	// notifications (so they never reach WebView2's native pipeline),
	// while NotificationReceived catches service-worker notifications
	// the shim cannot see. No overlap, no double toasts.
	view.OnNotification(func(title, body string) {
		a.backend.Dispatch(func() { a.notify(id, title, body) })
	})
	view.InitScript(notificationShim + "\n" + iconResolver)
	view.OnTitleChanged(func(title string) {
		a.backend.Dispatch(func() { a.onTitle(id, title) })
	})
	view.OnMessage(func(raw string) {
		a.backend.Dispatch(func() { a.onServiceMessage(id, raw) })
	})
	view.OnFaviconChanged(func(png []byte) {
		a.backend.Dispatch(func() { a.onFavicon(id, png) })
	})
	// Ordinary links leave for the default browser; same-site and OAuth
	// popups keep WebView2's default window (same profile, so logins
	// work). The decision must be synchronous — no Dispatch here.
	serviceURL := svc.URL
	view.OnNewWindow(func(url string) bool {
		if !openExternally(serviceURL, url) {
			return false
		}
		a.backend.OpenURL(url)
		return true
	})
	visible := id == a.activeID && !a.settingsOpen
	view.SetVisible(visible)
	view.SetMemoryTargetLow(!visible)
	if svc.URL == "" {
		view.NavigateHTML(testServicePage)
	} else {
		view.Navigate(svc.URL)
	}
	a.views[id] = view
	return nil
}

// applyLayout positions the chrome and every content view. Hidden views
// get the content bounds too, so switching never shows a stale size.
func (a *App) applyLayout() {
	if a.width == 0 || a.height == 0 {
		return
	}
	l := ComputeLayout(a.width, a.height, a.dpi, a.collapsed)
	if a.chrome != nil {
		a.chrome.SetBounds(l.Sidebar)
	}
	if a.settings != nil {
		a.settings.SetBounds(l.Content)
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
		a.closeSettings()
		a.activate(ctx, m.ServiceID)
	case bridge.OpenSettings:
		a.openSettings(ctx)
	case bridge.ToggleSidebar:
		a.collapsed = !a.collapsed
		a.applyLayout()
		a.renderChrome()
		v := "false"
		if a.collapsed {
			v = "true"
		}
		if err := a.store.PutSetting(ctx, store.PutSettingParams{Key: settingCollapsed, Value: v}); err != nil {
			log.Printf("store: save sidebar state: %v", err)
		}
	}
}

func (a *App) onServiceMessage(id int64, raw string) {
	msg, err := bridge.Decode(raw)
	if err != nil {
		log.Printf("service %d: %v", id, err)
		return
	}
	switch m := msg.(type) {
	case bridge.Notify:
		a.notify(id, m.Title, m.Body)
	case bridge.Icon:
		const maxIcon = 512 * 1024
		if !strings.HasPrefix(m.URI, "data:image/") || len(m.URI) > maxIcon {
			return
		}
		// Page-resolved icons beat the coarse WebView2 favicon.
		a.shimIcon[id] = true
		a.saveIcon(id, []byte(m.URI))
	}
}

// notify raises a native notification for a service, with its icon.
func (a *App) notify(id int64, title, body string) {
	log.Printf("app: notify from service %d: %q", id, title)
	a.lastNotifyID = id
	var icon image.Image
	if svc, ok := a.serviceByID(id); ok {
		if title == "" {
			title = svc.Name
		}
		icon = decodeIcon(svc.Favicon)
	}
	a.win.Tray().Notify(title, body, icon)
}

// decodeIcon turns a stored service icon (raw PNG, or a data URI from
// the in-page resolver) into an image; nil when it doesn't decode.
func decodeIcon(stored []byte) image.Image {
	raw := stored
	if bytes.HasPrefix(stored, []byte("data:")) {
		_, b64, ok := bytes.Cut(stored, []byte(";base64,"))
		if !ok {
			return nil
		}
		decoded, err := base64.StdEncoding.AppendDecode(nil, b64)
		if err != nil {
			return nil
		}
		raw = decoded
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	return img
}

// onFavicon handles WebView2's own favicon event — the 16px fallback,
// used only until the page-resolved icon arrives.
func (a *App) onFavicon(id int64, png []byte) {
	if a.shimIcon[id] {
		return
	}
	a.saveIcon(id, png)
}

// saveIcon stores a changed service icon and refreshes the sidebar.
func (a *App) saveIcon(id int64, icon []byte) {
	for i := range a.services {
		if a.services[i].ID != id || bytes.Equal(a.services[i].Favicon, icon) {
			continue
		}
		a.services[i].Favicon = icon
		if id > 0 { // synthetic services (test) are not persisted
			ctx := context.Background()
			if err := a.store.SetServiceFavicon(ctx, store.SetServiceFaviconParams{Favicon: icon, ID: id}); err != nil {
				log.Printf("store: save favicon: %v", err)
			}
		}
		a.renderChrome()
		return
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
		prev.SetMemoryTargetLow(true)
	}
	a.activeID = id
	view := a.views[id]
	view.SetMemoryTargetLow(false)
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
			Active: svc.ID == a.activeID && !a.settingsOpen,
			Badge:  a.badges[svc.ID],
			Icon:   svc.Favicon,
		}
	}
	return chrome.State{Items: items, SettingsOpen: a.settingsOpen, Collapsed: a.collapsed}
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

// openSettings shows the settings view.
func (a *App) openSettings(ctx context.Context) {
	if a.settingsOpen {
		return
	}
	if active, ok := a.views[a.activeID]; ok {
		active.SetVisible(false)
		active.SetMemoryTargetLow(true)
	}
	a.settings.Resume()
	a.settings.SetVisible(true)
	a.settings.Focus()
	a.settingsOpen = true
	a.renderChrome()
	a.renderSettings(ctx)
}

// closeSettings hides the settings view and restores the active service.
func (a *App) closeSettings() {
	if !a.settingsOpen {
		return
	}
	a.settings.SetVisible(false)
	// Nothing runs in the background of the settings page; sleep it.
	a.settings.Suspend()
	if active, ok := a.views[a.activeID]; ok {
		active.SetMemoryTargetLow(false)
		active.SetVisible(true)
		active.Focus()
	}
	a.settingsOpen = false
	a.renderChrome()
}

func (a *App) onSettingsMessage(ctx context.Context, raw string) {
	msg, err := bridge.Decode(raw)
	if err != nil {
		log.Printf("settings: %v", err)
		return
	}
	switch m := msg.(type) {
	case bridge.Ready:
		a.renderSettings(ctx)
	case bridge.CloseSettings:
		a.closeSettings()
	case bridge.AddService:
		a.settingsErr = errText(a.addService(ctx, m))
	case bridge.RemoveService:
		a.settingsErr = errText(a.removeService(ctx, m.ServiceID))
	case bridge.ToggleService:
		a.settingsErr = errText(a.toggleService(ctx, m))
	case bridge.MoveService:
		a.settingsErr = errText(a.moveService(ctx, m))
	case bridge.SetBadgeRegex:
		a.settingsErr = errText(a.setBadgeRegex(ctx, m))
	default:
		return
	}
	if _, isReady := msg.(bridge.Ready); !isReady {
		a.renderSettings(ctx)
	}
}

func errText(err error) string {
	if err != nil {
		log.Printf("settings: %v", err)
		return err.Error()
	}
	return ""
}

func (a *App) addService(ctx context.Context, m bridge.AddService) error {
	name, url := strings.TrimSpace(m.Name), strings.TrimSpace(m.URL)
	if m.Preset != "" {
		p, ok := service.PresetByKey(m.Preset)
		if !ok {
			return fmt.Errorf("unknown preset %q", m.Preset)
		}
		name, url = p.Name, p.URL
	}
	if name == "" {
		return fmt.Errorf("a service needs a name")
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("the URL must start with http:// or https://")
	}
	profiles, err := a.store.ListProfiles(ctx)
	if err != nil {
		return err
	}
	all, err := a.store.ListAllServices(ctx)
	if err != nil {
		return err
	}
	_, err = a.store.CreateService(ctx, store.CreateServiceParams{
		Preset:   m.Preset,
		Name:     name,
		Url:      url,
		Profile:  service.NewProfile(name, profiles),
		Position: int64(len(all)),
	})
	if err != nil {
		return err
	}
	return a.reload(ctx)
}

func (a *App) removeService(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("the test service cannot be removed")
	}
	// Removal is permanent: take the browsing profile (cookies, logins)
	// with it. The mark takes effect when reload closes the webview.
	// Disabling a service keeps the profile, by contrast.
	if view, ok := a.views[id]; ok {
		view.DeleteProfile()
	}
	if err := a.store.DeleteService(ctx, id); err != nil {
		return err
	}
	return a.reload(ctx)
}

func (a *App) toggleService(ctx context.Context, m bridge.ToggleService) error {
	if m.ServiceID <= 0 {
		return fmt.Errorf("the test service cannot be disabled")
	}
	enabled := int64(0)
	if m.Enabled {
		enabled = 1
	}
	if err := a.store.SetServiceEnabled(ctx, store.SetServiceEnabledParams{Enabled: enabled, ID: m.ServiceID}); err != nil {
		return err
	}
	return a.reload(ctx)
}

func (a *App) moveService(ctx context.Context, m bridge.MoveService) error {
	rows, err := a.store.ListAllServices(ctx)
	if err != nil {
		return err
	}
	i := -1
	for n, r := range rows {
		if r.ID == m.ServiceID {
			i = n
		}
	}
	j := i + m.Delta
	if i < 0 || j < 0 || j >= len(rows) {
		return nil
	}
	rows[i], rows[j] = rows[j], rows[i]
	// Renumber the whole list; it is small and this keeps positions dense.
	for n, r := range rows {
		if err := a.store.UpdateServicePosition(ctx, store.UpdateServicePositionParams{Position: int64(n), ID: r.ID}); err != nil {
			return err
		}
	}
	return a.reload(ctx)
}

func (a *App) setBadgeRegex(ctx context.Context, m bridge.SetBadgeRegex) error {
	regex := strings.TrimSpace(m.Regex)
	if regex != "" {
		if _, err := badge.Compile(regex); err != nil {
			return fmt.Errorf("bad pattern: %v", err)
		}
	}
	if m.ServiceID <= 0 {
		return fmt.Errorf("the test service keeps the generic rule")
	}
	if err := a.store.SetServiceBadgeRegex(ctx, store.SetServiceBadgeRegexParams{BadgeRegex: regex, ID: m.ServiceID}); err != nil {
		return err
	}
	return a.reload(ctx)
}

// reload re-reads the store and reconciles the running webviews with it:
// new/re-enabled services get views, removed/disabled ones are closed.
// The sidebar and settings page re-render from the result.
func (a *App) reload(ctx context.Context) error {
	services, err := a.store.Services(ctx)
	if err != nil {
		return err
	}
	a.services = a.withTestService(services)

	current := map[int64]service.Service{}
	for _, svc := range a.services {
		current[svc.ID] = svc
		a.rules[svc.ID] = a.ruleFor(svc)
	}
	for id, view := range a.views {
		if _, ok := current[id]; !ok {
			view.Close()
			delete(a.views, id)
			delete(a.badges, id)
			delete(a.rules, id)
		}
	}
	if _, ok := current[a.activeID]; !ok && len(a.services) > 0 {
		a.activeID = a.services[0].ID
	}
	for _, svc := range a.services {
		if _, ok := a.views[svc.ID]; !ok {
			if err := a.addServiceView(svc); err != nil {
				return err
			}
		}
	}
	a.applyLayout()
	a.renderChrome()
	return nil
}

func (a *App) renderSettings(ctx context.Context) {
	if a.settings == nil {
		return
	}
	state, err := a.settingsState(ctx)
	if err != nil {
		log.Printf("settings: %v", err)
		return
	}
	html, err := chrome.RenderSettings(state)
	if err != nil {
		log.Printf("settings: render: %v", err)
		return
	}
	msg, err := bridge.Encode(bridge.Render{HTML: html})
	if err != nil {
		log.Printf("settings: encode: %v", err)
		return
	}
	a.settings.PostJSON(msg)
}

func (a *App) settingsState(ctx context.Context) (chrome.SettingsState, error) {
	rows, err := a.store.ListAllServices(ctx)
	if err != nil {
		return chrome.SettingsState{}, err
	}
	items := make([]chrome.SettingsItem, len(rows))
	for i, r := range rows {
		items[i] = chrome.SettingsItem{
			ID:         r.ID,
			Name:       r.Name,
			URL:        r.Url,
			Enabled:    r.Enabled != 0,
			BadgeRegex: r.BadgeRegex,
			First:      i == 0,
			Last:       i == len(rows)-1,
		}
	}
	presets := make([]chrome.SettingsPreset, len(service.Presets))
	for i, p := range service.Presets {
		presets[i] = chrome.SettingsPreset{Key: p.Key, Name: p.Name}
	}
	return chrome.SettingsState{Items: items, Presets: presets, Error: a.settingsErr}, nil
}

// withTestService appends the debug-only Test service: synthetic, never
// persisted; an empty URL means the built-in test page, and the negative
// ID keeps clear of database rows. It borrows a stored icon so test
// notifications exercise the icon path.
func (a *App) withTestService(services []service.Service) []service.Service {
	if !a.debug {
		return services
	}
	test := service.Service{ID: -1, Name: "Test", Profile: "svc-test"}
	for _, svc := range services {
		if len(svc.Favicon) > 0 {
			test.Favicon = svc.Favicon
			break
		}
	}
	return append(services, test)
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
	if a.settings != nil {
		a.settings.Close()
	}
	a.backend.Quit()
}

func jsonInt(v int64) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
