package bridge

import "testing"

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{"render", Render{HTML: "<nav>hi</nav>"}},
		{"ready", Ready{}},
		{"activate", Activate{ServiceID: 42}},
		{"notify", Notify{Title: "WhatsApp", Body: "hello"}},
		{"badge", BadgeUpdate{Count: 3, Dot: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := Encode(tt.msg)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			got, err := Decode(raw)
			if err != nil {
				t.Fatalf("Decode(%s): %v", raw, err)
			}
			if got != tt.msg {
				t.Fatalf("round trip: got %#v, want %#v", got, tt.msg)
			}
		})
	}
}

func TestDecodePageMessages(t *testing.T) {
	// Messages exactly as the glue script and shim post them.
	got, err := Decode(`{"type":"activate","data":{"serviceId":3}}`)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := got.(Activate); !ok || a.ServiceID != 3 {
		t.Fatalf("got %#v", got)
	}
	if _, err := Decode(`{"type":"ready"}`); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeRejectsUnknown(t *testing.T) {
	if _, err := Decode(`{"type":"drop-tables"}`); err == nil {
		t.Fatal("unknown type must error")
	}
	if _, err := Decode(`not json`); err == nil {
		t.Fatal("bad JSON must error")
	}
	if _, err := Encode(struct{}{}); err == nil {
		t.Fatal("unknown message type must error")
	}
}
