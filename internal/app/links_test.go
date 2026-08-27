package app

import "testing"

func TestOpenExternally(t *testing.T) {
	const slack = "https://app.slack.com/client"
	tests := []struct {
		service, target string
		want            bool
	}{
		// Ordinary links leave the app.
		{slack, "https://example.com/article", true},
		{slack, "https://news.ycombinator.com/item?id=1", true},
		// Same site stays (workspace popups, file views).
		{slack, "https://files.slack.com/f/abc", false},
		{slack, "https://slack.com/help", false},
		// Identity providers stay, or logins break.
		{slack, "https://accounts.google.com/o/oauth2/auth", false},
		{slack, "https://login.microsoftonline.com/common", false},
		// Non-web schemes stay internal.
		{slack, "about:blank", false},
		{slack, "", false},
		{"https://web.whatsapp.com", "https://whatsapp.com/download", false},
		{"https://web.whatsapp.com", "https://faq.whatsapp.com/x", false},
	}
	for _, tt := range tests {
		if got := openExternally(tt.service, tt.target); got != tt.want {
			t.Errorf("openExternally(%q, %q) = %v, want %v", tt.service, tt.target, got, tt.want)
		}
	}
}
