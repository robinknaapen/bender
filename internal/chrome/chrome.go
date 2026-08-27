// Package chrome renders bender's shell UI — the sidebar — with loom.
// Render is a pure function from State to an HTML fragment; the app calls
// it on every state change and pushes the result to the chrome webview,
// which swaps it in. All UI state lives on the Go side.
package chrome

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/a-h/templ"
	"github.com/pietjan/loom"

	"github.com/pietjan/bender/internal/badge"
)

// State is everything the sidebar shows.
type State struct {
	Items        []Item
	SettingsOpen bool
}

// Item is one service entry.
type Item struct {
	ID     int64
	Name   string
	Active bool
	Badge  badge.Badge
}

// SettingsState is everything the settings page shows.
type SettingsState struct {
	Items   []SettingsItem
	Presets []SettingsPreset
	// Error is a validation message from the last rejected action.
	Error string
}

// SettingsItem is one configured service row.
type SettingsItem struct {
	ID          int64
	Name        string
	URL         string
	Enabled     bool
	BadgeRegex  string
	First, Last bool
}

// SettingsPreset is one addable built-in service.
type SettingsPreset struct {
	Key  string
	Name string
}

// Render produces the sidebar fragment for state.
func Render(state State) (string, error) {
	return render(sidebar(state))
}

// RenderSettings produces the settings page fragment.
func RenderSettings(state SettingsState) (string, error) {
	return render(settings(state))
}

func render(c templ.Component) (string, error) {
	var sb strings.Builder
	// A fresh loom context per render keeps generated element IDs
	// deterministic, so identical states produce identical HTML.
	ctx := loom.NewContext(context.Background())
	if err := c.Render(ctx, &sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// msgAttr builds the JSON bridge message an element posts when clicked.
func msgAttr(msgType string, data map[string]any) string {
	msg := map[string]any{"type": msgType}
	if data != nil {
		msg["data"] = data
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		panic(err) // static shapes; cannot fail
	}
	return string(raw)
}

// initials derives up to two letters for a service avatar.
func initials(name string) string {
	fields := strings.Fields(name)
	var b strings.Builder
	for i, f := range fields {
		if i == 2 {
			break
		}
		r := []rune(f)
		b.WriteString(strings.ToUpper(string(r[0])))
	}
	if b.Len() == 0 {
		return "?"
	}
	return b.String()
}
