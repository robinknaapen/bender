package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pietjan/bender/internal/service"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "bender.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMigrateAndSeed(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if err := s.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	services, err := s.Services(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != len(service.Presets) {
		t.Fatalf("seeded %d services, want %d", len(services), len(service.Presets))
	}
	if services[0].Name != "WhatsApp" || services[0].Profile != "svc-whatsapp" {
		t.Fatalf("first service %+v", services[0])
	}
	// Seeding twice must not duplicate.
	if err := s.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountServices(ctx); n != int64(len(service.Presets)) {
		t.Fatalf("re-seed duplicated: %d rows", n)
	}
}

func TestSettingsUpsert(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if _, err := s.GetSetting(ctx, "missing"); err == nil {
		t.Fatal("missing setting must error")
	}
	for _, v := range []string{"a", "b"} {
		if err := s.PutSetting(ctx, PutSettingParams{Key: "k", Value: v}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.GetSetting(ctx, "k")
	if err != nil || got != "b" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDisabledServicesHidden(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if err := s.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	all, _ := s.Services(ctx)
	if err := s.SetServiceEnabled(ctx, SetServiceEnabledParams{Enabled: 0, ID: all[0].ID}); err != nil {
		t.Fatal(err)
	}
	left, _ := s.Services(ctx)
	if len(left) != len(all)-1 {
		t.Fatalf("disabled service still listed: %d of %d", len(left), len(all))
	}
}
