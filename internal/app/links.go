package app

import (
	"net/url"
	"strings"
)

// authSites are identity-provider sites whose popups must stay in-app —
// an OAuth window opened in the external browser has the wrong cookie
// jar and the login never completes.
var authSites = []string{
	"google.com",
	"live.com",
	"microsoft.com",
	"microsoftonline.com",
	"apple.com",
	"facebook.com",
	"okta.com",
	"auth0.com",
}

// openExternally reports whether a popup from a service page belongs in
// the default browser. Same-site targets and identity providers stay
// in-app (login flows); everything else — links people sent you — is
// ordinary web content and leaves.
func openExternally(serviceURL, target string) bool {
	t, err := url.Parse(target)
	if err != nil || (t.Scheme != "http" && t.Scheme != "https") {
		return false // about:blank and friends stay internal
	}
	if s, err := url.Parse(serviceURL); err == nil && sameSite(s.Hostname(), t.Hostname()) {
		return false
	}
	for _, site := range authSites {
		if siteOf(t.Hostname()) == site {
			return false
		}
	}
	return true
}

// sameSite reports whether two hosts share a registrable domain,
// approximated as the last two labels ("web.whatsapp.com" → "whatsapp.com").
func sameSite(a, b string) bool {
	return siteOf(a) != "" && siteOf(a) == siteOf(b)
}

func siteOf(host string) string {
	labels := strings.Split(strings.ToLower(strings.TrimSuffix(host, ".")), ".")
	if len(labels) < 2 {
		return strings.ToLower(host)
	}
	return strings.Join(labels[len(labels)-2:], ".")
}
