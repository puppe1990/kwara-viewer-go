package db

import (
	"testing"

	"github.com/puppe1990/kwara-viewer/internal/store"
)

func TestRunSeeds_InsertsMonitorLotsOnce(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := RunSeeds(s); err != nil {
		t.Fatal(err)
	}
	first, err := s.CountLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if first < 50 {
		t.Fatalf("seeded %d, want at least 50 monitor lots", first)
	}

	if err := RunSeeds(s); err != nil {
		t.Fatal(err)
	}
	second, err := s.CountLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("second seed changed count %d -> %d", first, second)
	}

	openSP, err := s.ListLots(store.LotFilter{PickupState: "SP", Query: "Dell"})
	if err != nil {
		t.Fatal(err)
	}
	if len(openSP) == 0 {
		t.Fatal("expected Dell lots in SP seed")
	}
}
