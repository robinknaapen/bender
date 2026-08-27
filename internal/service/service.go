// Package service holds the service domain type and the built-in presets.
// A preset carries what ships with the app (URL, badge behaviour); a
// Service row in the store carries what the user chose.
package service

// Service is one configured messaging service.
type Service struct {
	ID      int64
	Preset  string // preset key; "" or unknown means custom
	Name    string
	URL     string
	Profile string // browsing-session identifier, unique per service
	// BadgeRegex, when non-empty, overrides the preset badge rule; its
	// first capture group is the unread count.
	BadgeRegex string
}

// Preset is a built-in service definition.
type Preset struct {
	Key  string
	Name string
	URL  string
}

// Presets are the built-in services, in the order they are seeded.
var Presets = []Preset{
	{Key: "whatsapp", Name: "WhatsApp", URL: "https://web.whatsapp.com"},
	{Key: "slack", Name: "Slack", URL: "https://app.slack.com/client"},
	{Key: "discord", Name: "Discord", URL: "https://discord.com/app"},
	{Key: "telegram", Name: "Telegram", URL: "https://web.telegram.org"},
	{Key: "messenger", Name: "Messenger", URL: "https://www.messenger.com"},
	{Key: "gmail", Name: "Gmail", URL: "https://mail.google.com"},
}

// PresetByKey returns the preset for key, if it exists.
func PresetByKey(key string) (Preset, bool) {
	for _, p := range Presets {
		if p.Key == key {
			return p, true
		}
	}
	return Preset{}, false
}
