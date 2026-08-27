package badge

import "testing"

func TestGeneric(t *testing.T) {
	tests := []struct {
		title   string
		prev    Badge
		want    Badge
		changed bool
	}{
		{"(3) WhatsApp", Badge{}, Badge{Count: 3}, true},
		{"(99+) Inbox", Badge{}, Badge{Count: 99}, true},
		{"WhatsApp", Badge{Count: 3}, Badge{}, true},
		{"", Badge{Count: 3}, Badge{Count: 3}, false}, // says nothing, keep
	}
	for _, tt := range tests {
		got, changed := Parse(Generic, tt.prev, tt.title)
		if got != tt.want || changed != tt.changed {
			t.Errorf("Parse(Generic, %v, %q) = %v,%v want %v,%v",
				tt.prev, tt.title, got, changed, tt.want, tt.changed)
		}
	}
}

func TestSlackDot(t *testing.T) {
	rule := ForPreset("slack")
	got, changed := Parse(rule, Badge{}, "* general - Slack")
	if !changed || !got.Dot {
		t.Fatalf("unread marker: got %v,%v", got, changed)
	}
	got, changed = Parse(rule, Badge{Dot: true}, "general - Slack")
	if !changed || !got.Zero() {
		t.Fatalf("clear: got %v,%v", got, changed)
	}
}

func TestForPresetFallsBack(t *testing.T) {
	if r := ForPreset("no-such-preset"); r.CountRe != Generic.CountRe {
		t.Fatal("unknown preset must fall back to Generic")
	}
}

func TestCompileOverride(t *testing.T) {
	rule, err := Compile(`\[(\d+) new\]`)
	if err != nil {
		t.Fatal(err)
	}
	got, changed := Parse(rule, Badge{}, "Inbox [7 new] - Custom")
	if !changed || got.Count != 7 {
		t.Fatalf("got %v,%v", got, changed)
	}
	// User patterns are total: a non-matching title clears.
	got, changed = Parse(rule, Badge{Count: 7}, "Inbox - Custom")
	if !changed || !got.Zero() {
		t.Fatalf("clear: got %v,%v", got, changed)
	}
	if _, err := Compile(`(`); err == nil {
		t.Fatal("bad pattern must error")
	}
}
