// Package chrome renders bender's shell UI — the sidebar — with loom.
// Render is a pure function from State to an HTML fragment; the app calls
// it on every state change and pushes the result to the chrome webview,
// which swaps it in. All UI state lives on the Go side.
package chrome

import (
	"context"
	"strings"

	"github.com/pietjan/loom"

	"github.com/pietjan/bender/internal/badge"
)

// State is everything the sidebar shows.
type State struct {
	Items []Item
}

// Item is one service entry.
type Item struct {
	ID     int64
	Name   string
	Active bool
	Badge  badge.Badge
}

// Render produces the sidebar fragment for state.
func Render(state State) (string, error) {
	var sb strings.Builder
	// A fresh loom context per render keeps generated element IDs
	// deterministic, so identical states produce identical HTML.
	ctx := loom.NewContext(context.Background())
	if err := sidebar(state).Render(ctx, &sb); err != nil {
		return "", err
	}
	return sb.String(), nil
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
