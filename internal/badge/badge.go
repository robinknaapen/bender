// Package badge turns document titles into unread indicators. Services
// encode unread state in their page title — "(3) WhatsApp", "* Slack" —
// and this package holds the parsing rules. Pure functions over strings.
package badge

import (
	"regexp"
	"strconv"
)

// Badge is the unread state of one service.
type Badge struct {
	// Count is the number of unread items, when the service exposes one.
	Count int
	// Dot marks unread-without-a-count (e.g. Slack's "*" prefix).
	Dot bool
}

// Zero reports whether the badge shows nothing.
func (b Badge) Zero() bool { return b.Count == 0 && !b.Dot }

// Rule extracts a Badge from a title. CountRe's first capture group is the
// unread count; DotRe, when set, marks unread without a count. ClearRe,
// when set, is the only pattern allowed to clear an existing badge — a
// title matching nothing leaves the badge unchanged, so transient titles
// ("Connecting…") never wipe real state.
type Rule struct {
	CountRe *regexp.Regexp
	DotRe   *regexp.Regexp
	ClearRe *regexp.Regexp
}

// Generic is the fallback rule: a leading parenthesised number is the
// count — "(3) WhatsApp", "(99+) Inbox" — and any title without one is
// considered read.
var Generic = Rule{
	CountRe: regexp.MustCompile(`^\((\d+)\+?\)`),
	ClearRe: regexp.MustCompile(`^[^(]`),
}

// rules per service preset; Parse falls back to Generic for unknown keys.
var presets = map[string]Rule{
	// Slack prefixes the title with "*" for unread and "! " for mentions;
	// no numeric count in the title.
	"slack": {
		DotRe:   regexp.MustCompile(`^[*!]`),
		ClearRe: regexp.MustCompile(`^[^*!]`),
	},
}

// ForPreset returns the rule for a service preset key.
func ForPreset(key string) Rule {
	if r, ok := presets[key]; ok {
		return r
	}
	return Generic
}

// Compile builds a Rule from a user-supplied count pattern, as stored in
// the services table. The pattern's first capture group is the count; a
// non-matching title clears the badge (user patterns are assumed total).
func Compile(pattern string) (Rule, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Rule{}, err
	}
	return Rule{CountRe: re, ClearRe: regexp.MustCompile(``)}, nil
}

// Parse applies rule to a title. changed is false when the title says
// nothing about unread state (no pattern matched), in which case prev
// should be kept.
func Parse(rule Rule, prev Badge, title string) (b Badge, changed bool) {
	if rule.CountRe != nil {
		if m := rule.CountRe.FindStringSubmatch(title); m != nil {
			n, err := strconv.Atoi(m[1])
			if err == nil {
				return Badge{Count: n}, true
			}
		}
	}
	if rule.DotRe != nil && rule.DotRe.MatchString(title) {
		return Badge{Dot: true}, true
	}
	if rule.ClearRe != nil && rule.ClearRe.MatchString(title) {
		return Badge{}, true
	}
	return prev, false
}
