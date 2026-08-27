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

// OpenSettings asks to show the settings view; CloseSettings dismisses it.
type OpenSettings struct{}
type CloseSettings struct{}

// Settings page → Go. Form-sourced fields arrive as strings (FormData).

// AddService adds a service from a preset key or a custom name+URL.
type AddService struct {
	Preset string `json:"preset"`
	Name   string `json:"name"`
	URL    string `json:"url"`
}

// RemoveService deletes a service permanently.
type RemoveService struct {
	ServiceID int64 `json:"serviceId"`
}

// ToggleService enables or disables a service.
type ToggleService struct {
	ServiceID int64 `json:"serviceId"`
	Enabled   bool  `json:"enabled"`
}

// MoveService moves a service up (-1) or down (+1) in the sidebar.
type MoveService struct {
	ServiceID int64 `json:"serviceId"`
	Delta     int   `json:"delta"`
}

// SetBadgeRegex sets (or clears) a service's badge-pattern override.
type SetBadgeRegex struct {
	ServiceID int64  `json:"serviceId,string"`
	Regex     string `json:"regex"`
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
	case "open-settings":
		v = &OpenSettings{}
	case "close-settings":
		v = &CloseSettings{}
	case "svc-add":
		v = &AddService{}
	case "svc-remove":
		v = &RemoveService{}
	case "svc-toggle":
		v = &ToggleService{}
	case "svc-move":
		v = &MoveService{}
	case "svc-badge":
		v = &SetBadgeRegex{}
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
	case OpenSettings:
		return "open-settings", nil
	case CloseSettings:
		return "close-settings", nil
	case AddService:
		return "svc-add", nil
	case RemoveService:
		return "svc-remove", nil
	case ToggleService:
		return "svc-toggle", nil
	case MoveService:
		return "svc-move", nil
	case SetBadgeRegex:
		return "svc-badge", nil
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
	case *OpenSettings:
		return *m
	case *CloseSettings:
		return *m
	case *AddService:
		return *m
	case *RemoveService:
		return *m
	case *ToggleService:
		return *m
	case *MoveService:
		return *m
	case *SetBadgeRegex:
		return *m
	}
	return v
}
