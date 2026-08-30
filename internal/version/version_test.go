package version

import "testing"

func TestGet(t *testing.T) {
	got := Get()
	if got.Version == "" {
		t.Fatal("expected non-empty version")
	}
	if got.Commit == "" {
		t.Fatal("expected non-empty commit")
	}
	if got.Date == "" {
		t.Fatal("expected non-empty date")
	}
}
