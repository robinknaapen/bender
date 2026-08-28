package app

import (
	_ "embed"
)

var (
	// notificationShim is injected into every service webview before any page
	// script runs. It replaces window.Notification with a stand-in that posts
	// the notification to Go (which raises a native one) and reports the
	// permission as granted, so sites don't gate on a prompt that native
	// WebView2 would never show usefully. Service-worker notifications bypass
	// this; the NotificationReceived COM event can cover those later.
	//go:embed js/notification.js
	notificationJS string

	// badgeSniffer is injected into every service webview. Some apps never
	// encode unread state in the document title (Mattermost shows "(n)" only
	// for mentions and nothing at all for plain unreads), so this reads it
	// from their DOM instead. Each reader detects its own app and returns
	// null elsewhere, which keeps the script safe to inject universally —
	// self-hosted instances included, where no preset key or URL can tell us
	// what the service is. State posts only on change; the host retires
	// title-based parsing for a service after its first badge message.

	//go:embed js/badge-sniffer.js
	badgeSnifferJS string

	// iconResolver is injected into every service webview.

	//go:embed js/icon-resolver.js
	iconResolverJS string
)
