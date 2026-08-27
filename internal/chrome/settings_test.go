package chrome

import (
	"strings"
	"testing"
)

func TestRenderSettings(t *testing.T) {
	state := SettingsState{
		Items: []SettingsItem{
			{ID: 1, Name: "WhatsApp", URL: "https://web.whatsapp.com", Enabled: true, First: true},
			{ID: 2, Name: "Slack", URL: "https://app.slack.com", Enabled: false, BadgeRegex: `\((\d+)\)`, Last: true},
		},
		Presets: []SettingsPreset{{Key: "discord", Name: "Discord"}},
		Error:   "the URL must start with http:// or https://",
	}
	html, err := RenderSettings(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`&#34;svc-remove&#34;`, // data-msg payloads survive attribute escaping
		`&#34;svc-toggle&#34;`,
		`&#34;svc-move&#34;`,
		`data-msg-type="svc-badge"`,
		`data-msg-type="svc-add"`,
		`&#34;preset&#34;:&#34;discord&#34;`,
		"the URL must start with",
		"WhatsApp",
		"Enable", // disabled Slack offers Enable
	} {
		if !strings.Contains(html, want) {
			t.Errorf("settings page missing %q", want)
		}
	}
}

func TestSidebarSettingsEntry(t *testing.T) {
	html, err := Render(State{Items: []Item{{ID: 1, Name: "WhatsApp", Active: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `&#34;open-settings&#34;`) {
		t.Fatal("sidebar missing the settings entry")
	}
	// With settings open, the settings item is current instead of a service.
	html, err = Render(State{Items: []Item{{ID: 1, Name: "WhatsApp"}}, SettingsOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(html, `aria-current="page"`) != 1 {
		t.Fatal("exactly one current item expected with settings open")
	}
}
