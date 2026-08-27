// Package bridge defines the JSON message contract between the Go core and
// the pages it hosts: the chrome (sidebar) webview and the notification shim
// injected into service webviews. Pure data and codecs; no I/O.
package bridge

import (
	"encoding/json"
	"fmt"
)

// Envelope is the wire format: a type tag and a type-specific payload.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Go → chrome.

// Render carries a freshly rendered sidebar document body. The chrome page
// swaps it into its mount point wholesale; all UI state lives in Go.
type Render struct {
	HTML string `json:"html"`
}

// Chrome → Go.

// Ready signals that the chrome page has loaded and wants its first Render.
type Ready struct{}

// Activate asks to switch to a service.
type Activate struct {
	ServiceID int64 `json:"serviceId"`
}

// Service shim → Go.

// Notify reports a web notification raised by a service page. The service
// is implied by which webview delivered the message.
type Notify struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Encode wraps a message in an Envelope and marshals it. The concrete type
// of v picks the type tag; passing any other type is a programming error.
func Encode(v any) (string, error) {
	tag, err := tagOf(v)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("bridge: marshal %s: %w", tag, err)
	}
	raw, err := json.Marshal(Envelope{Type: tag, Data: data})
	if err != nil {
		return "", fmt.Errorf("bridge: marshal envelope: %w", err)
	}
	return string(raw), nil
}

// Decode parses an envelope and returns the concrete message value:
// Ready, Activate, or Notify. Unknown types are an error, not a panic —
// the page side is the untrusted half of this conversation.
func Decode(raw string) (any, error) {
	var env Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil, fmt.Errorf("bridge: bad envelope: %w", err)
	}
	var v any
	switch env.Type {
	case "ready":
		v = &Ready{}
	case "activate":
		v = &Activate{}
	case "notify":
		v = &Notify{}
	case "render":
		v = &Render{}
	default:
		return nil, fmt.Errorf("bridge: unknown message type %q", env.Type)
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, v); err != nil {
			return nil, fmt.Errorf("bridge: bad %s payload: %w", env.Type, err)
		}
	}
	return deref(v), nil
}

func tagOf(v any) (string, error) {
	switch v.(type) {
	case Render:
		return "render", nil
	case Ready:
		return "ready", nil
	case Activate:
		return "activate", nil
	case Notify:
		return "notify", nil
	default:
		return "", fmt.Errorf("bridge: unknown message %T", v)
	}
}

func deref(v any) any {
	switch m := v.(type) {
	case *Render:
		return *m
	case *Ready:
		return *m
	case *Activate:
		return *m
	case *Notify:
		return *m
	}
	return v
}
