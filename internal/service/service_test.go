package service

import "testing"

func TestNewProfile(t *testing.T) {
	tests := []struct {
		name  string
		taken []string
		want  string
	}{
		{"WhatsApp", nil, "svc-whatsapp"},
		{"Google Chat", nil, "svc-google-chat"},
		{"WhatsApp", []string{"svc-whatsapp"}, "svc-whatsapp-2"},
		{"WhatsApp", []string{"svc-whatsapp", "svc-whatsapp-2"}, "svc-whatsapp-3"},
		{"№±!", nil, "svc-custom"},
	}
	for _, tt := range tests {
		if got := NewProfile(tt.name, tt.taken); got != tt.want {
			t.Errorf("NewProfile(%q, %v) = %q, want %q", tt.name, tt.taken, got, tt.want)
		}
	}
}
