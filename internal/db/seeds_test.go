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

	embu, err := s.ListLots(store.LotFilter{Query: "Embu das Artes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(embu) == 0 {
		t.Fatal("expected Embu das Artes pickup city on K-3091 seed lots")
	}
	if embu[0].PickupAddress == "" {
		t.Fatal("expected pickup address on K-3091 seed lots")
	}
}

func TestRunSeeds_BackfillsPickupOnExistingLots(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	existing := store.Lot{
		ID:          "1745673805868566236",
		Title:       "Monitor gamer 27 AOC",
		KwaraID:     "K-3091",
		PickupState: "SP",
	}
	if err := s.UpsertLot(existing); err != nil {
		t.Fatal(err)
	}
	if err := RunSeeds(s); err != nil {
		t.Fatal(err)
	}
	n, err := s.CountLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1 (skip insert on nonempty table)", n)
	}
	got, err := s.ListLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].PickupCity != "Embu das Artes" {
		t.Errorf("city = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Eliza - Embu das Artes/SP" {
		t.Errorf("address = %q", got[0].PickupAddress)
	}
}
