package chrome

import (
	"strings"
	"testing"

	"github.com/pietjan/bender/internal/badge"
)

func testState() State {
	return State{Items: []Item{
		{ID: 1, Name: "WhatsApp", Active: true, Badge: badge.Badge{Count: 3}},
		{ID: 2, Name: "Slack", Badge: badge.Badge{Dot: true}},
		{ID: 3, Name: "Discord"},
	}}
}

func TestRender(t *testing.T) {
	html, err := Render(testState())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-service-id="1"`,
		`data-service-id="3"`,
		`aria-current="page"`,
		">3<",      // count badge
		"WhatsApp", // names
		"Discord",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered sidebar missing %q\n%s", want, html)
		}
	}
	if strings.Count(html, `aria-current="page"`) != 1 {
		t.Error("exactly one item may be current")
	}
}

func TestRenderDeterministic(t *testing.T) {
	a, err := Render(testState())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render(testState())
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("Render must be a pure function of State")
	}
}

func TestShellEmbedsGlue(t *testing.T) {
	shell := Shell()
	for _, want := range []string{"<!DOCTYPE html>", `id="mount"`, "postMessage"} {
		if !strings.Contains(shell, want) {
			t.Errorf("shell missing %q", want)
		}
	}
}

func TestInitials(t *testing.T) {
	tests := map[string]string{
		"WhatsApp":    "W",
		"Google Chat": "GC",
		"a b c":       "AB",
		"":            "?",
	}
	for in, want := range tests {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, want %q", in, got, want)
		}
	}
}
