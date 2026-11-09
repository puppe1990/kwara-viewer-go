package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais/fakedata"
)

func fakeLot(id string) Lot {
	title := "Monitor Dell " + fakedata.Title()
	return Lot{
		ID:               id,
		Number:           "020",
		Title:            title,
		Slug:             "monitor-dell-" + id,
		ListingID:        "listing-" + id,
		ListingTitle:     "Informática " + fakedata.Name(),
		KwaraID:          "K-3126",
		CurrentBidCents:  18000,
		StartingBidCents: 10000,
		BidsCount:        1,
		EndsAt:           "2026-11-09T20:04:45Z",
		PickupState:      "SP",
		Ended:            false,
		URL:              "https://www.kwara.com.br/bens/monitor-dell-" + id,
	}
}

func TestSQLiteStore_UpsertLotThenListsIt(t *testing.T) {
	fakedata.Seed(42)
	s := newTestStore(t)
	lot := fakeLot("1739237630056335265")

	if err := s.UpsertLot(lot); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := s.ListLots(LotFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	row := got[0]
	if row.ID != lot.ID {
		t.Errorf("id = %q, want %q", row.ID, lot.ID)
	}
	if row.Title != lot.Title {
		t.Errorf("title = %q, want %q", row.Title, lot.Title)
	}
	if row.KwaraID != "K-3126" {
		t.Errorf("kwaraId = %q", row.KwaraID)
	}
	if row.CurrentBidCents != 18000 {
		t.Errorf("bid = %d, want 18000", row.CurrentBidCents)
	}
	if row.PickupState != "SP" {
		t.Errorf("pickup = %q", row.PickupState)
	}
	if row.URL != lot.URL {
		t.Errorf("url = %q", row.URL)
	}
}

func TestSQLiteStore_UpsertLotPersistsPickupCityAndAddress(t *testing.T) {
	fakedata.Seed(43)
	s := newTestStore(t)
	lot := fakeLot("1745009854700848771")
	lot.PickupCity = "Taboão da Serra"
	lot.PickupAddress = "Jardim Triangulo - Taboão da Serra/SP"

	if err := s.UpsertLot(lot); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListLots(LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].PickupCity != "Taboão da Serra" {
		t.Errorf("city = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Triangulo - Taboão da Serra/SP" {
		t.Errorf("address = %q", got[0].PickupAddress)
	}
}

func TestSQLiteStore_PatchPickupFillsEmptyColumns(t *testing.T) {
	fakedata.Seed(44)
	s := newTestStore(t)
	if err := s.UpsertLot(fakeLot("1")); err != nil {
		t.Fatal(err)
	}
	if err := s.PatchPickup("1", "Taboão da Serra", "Jardim Triangulo - Taboão da Serra/SP", "SP"); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListLots(LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].PickupCity != "Taboão da Serra" {
		t.Errorf("city = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Triangulo - Taboão da Serra/SP" {
		t.Errorf("address = %q", got[0].PickupAddress)
	}

	if err := s.PatchPickup("1", "Outra", "Outro local", "RJ"); err != nil {
		t.Fatal(err)
	}
	got, err = s.ListLots(LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].PickupCity != "Taboão da Serra" {
		t.Errorf("city overwritten = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Triangulo - Taboão da Serra/SP" {
		t.Errorf("address overwritten = %q", got[0].PickupAddress)
	}
	if got[0].PickupState != "SP" {
		t.Errorf("state overwritten = %q", got[0].PickupState)
	}
}

func TestSQLiteStore_ListLotsQueryMatchesPickupCity(t *testing.T) {
	fakedata.Seed(45)
	s := newTestStore(t)
	lot := fakeLot("1")
	lot.PickupCity = "Taboão da Serra"
	lot.PickupAddress = "Jardim Triangulo - Taboão da Serra/SP"
	if err := s.UpsertLot(lot); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListLots(LotFilter{Query: "Taboão"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("query city = %+v", idsOf(got))
	}
}

func TestSQLiteStore_UpsertLotUpdatesCurrentBid(t *testing.T) {
	fakedata.Seed(7)
	s := newTestStore(t)
	lot := fakeLot("99")
	if err := s.UpsertLot(lot); err != nil {
		t.Fatal(err)
	}
	lot.CurrentBidCents = 25000
	lot.BidsCount = 4
	if err := s.UpsertLot(lot); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListLots(LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (upsert not insert)", len(got))
	}
	if got[0].CurrentBidCents != 25000 {
		t.Errorf("bid = %d, want 25000", got[0].CurrentBidCents)
	}
	if got[0].BidsCount != 4 {
		t.Errorf("bids = %d, want 4", got[0].BidsCount)
	}
}

func TestSQLiteStore_ListLotsFiltersQueryPickupAndOpen(t *testing.T) {
	fakedata.Seed(11)
	s := newTestStore(t)
	openSP := fakeLot("1")
	openSP.Title = "Monitor Dell P2219H"
	openSP.PickupState = "SP"
	openSP.EndsAt = "2026-12-01T15:00:00Z"
	openSP.Ended = false

	endedSP := fakeLot("2")
	endedSP.Title = "Monitor LG ultrawide"
	endedSP.PickupState = "SP"
	endedSP.Ended = true
	endedSP.EndsAt = "2026-10-01T15:00:00Z"

	openMG := fakeLot("3")
	openMG.Title = "Monitor AOC 24"
	openMG.PickupState = "MG"
	openMG.Ended = false
	openMG.EndsAt = "2026-12-02T15:00:00Z"

	pastSP := fakeLot("4")
	pastSP.Title = "Monitor Samsung"
	pastSP.PickupState = "SP"
	pastSP.Ended = false
	pastSP.EndsAt = "2026-10-20T15:00:00Z"

	for _, lot := range []Lot{openSP, endedSP, openMG, pastSP} {
		if err := s.UpsertLot(lot); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Date(2026, 11, 9, 16, 0, 0, 0, time.UTC)

	dell, err := s.ListLots(LotFilter{Query: "dell", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(dell) != 1 || dell[0].ID != "1" {
		t.Fatalf("query dell = %+v", idsOf(dell))
	}

	sp, err := s.ListLots(LotFilter{PickupState: "SP", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(sp) != 3 {
		t.Fatalf("SP len = %d, ids=%v", len(sp), idsOf(sp))
	}

	open, err := s.ListLots(LotFilter{PickupState: "SP", OpenOnly: true, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].ID != "1" {
		t.Fatalf("open SP = %v, want [1]", idsOf(open))
	}
}

func TestSQLiteStore_ListLotsOpenFirstThenEndsAt(t *testing.T) {
	fakedata.Seed(3)
	s := newTestStore(t)
	later := fakeLot("later")
	later.Ended = false
	later.EndsAt = "2026-12-10T12:00:00Z"
	sooner := fakeLot("sooner")
	sooner.Ended = false
	sooner.EndsAt = "2026-12-01T12:00:00Z"
	closed := fakeLot("closed")
	closed.Ended = true
	closed.EndsAt = "2026-11-01T12:00:00Z"
	for _, lot := range []Lot{later, closed, sooner} {
		if err := s.UpsertLot(lot); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListLots(LotFilter{Now: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sooner", "later", "closed"}
	if ids := idsOf(got); len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("order = %v, want %v", ids, want)
	}
}

func TestSQLiteStore_ListLotsPaginates(t *testing.T) {
	s := newTestStore(t)
	seedNumberedLots(t, s, 11)

	page1, err := s.ListLots(LotFilter{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 10 {
		t.Fatalf("page1 len = %d, want 10", len(page1))
	}
	page2, err := s.ListLots(LotFilter{Limit: 10, Offset: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2))
	}
	if page2[0].ID != "11" {
		t.Errorf("page2 id = %q, want 11", page2[0].ID)
	}
	if idsOf(page1)[0] != "01" {
		t.Errorf("page1 first = %v", idsOf(page1))
	}
}

func TestSQLiteStore_ListLotsSortsByBidDesc(t *testing.T) {
	s := newTestStore(t)
	seedNumberedLots(t, s, 11)

	got, err := s.ListLots(LotFilter{Sort: "bid", Desc: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "11" || got[0].CurrentBidCents != 11000 {
		t.Fatalf("first = id %s bid %d, want 11 / 11000", got[0].ID, got[0].CurrentBidCents)
	}
	if got[1].ID != "10" {
		t.Errorf("second = %s", got[1].ID)
	}
}

func TestSQLiteStore_ListLotsSortsByCity(t *testing.T) {
	s := newTestStore(t)
	rows := []Lot{fakeLot("a"), fakeLot("b"), fakeLot("c")}
	rows[0].PickupCity = "Taboão da Serra"
	rows[1].PickupCity = "São Paulo"
	rows[2].PickupCity = "Embu das Artes"
	for i := range rows {
		rows[i].Title = "Monitor " + rows[i].PickupCity
		rows[i].EndsAt = "2026-12-01T15:00:00Z"
		if err := s.UpsertLot(rows[i]); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListLots(LotFilter{Sort: "city"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c", "b", "a"}
	if ids := idsOf(got); len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("city order = %v, want %v", ids, want)
	}
}

func seedNumberedLots(t *testing.T, s *SQLiteStore, n int) {
	t.Helper()
	cities := []string{"Embu das Artes", "São Paulo", "Taboão da Serra"}
	for i := 1; i <= n; i++ {
		lot := fakeLot(fmt.Sprintf("%02d", i))
		lot.Number = fmt.Sprintf("%03d", i)
		lot.Title = fmt.Sprintf("Monitor lote %02d", i)
		lot.KwaraID = fmt.Sprintf("K-%d", 3000+i)
		lot.CurrentBidCents = int64(i * 1000)
		lot.PickupCity = cities[(i-1)%len(cities)]
		lot.EndsAt = "2026-12-01T15:00:00Z"
		lot.Ended = false
		if err := s.UpsertLot(lot); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSQLiteStore_UpsertLotRejectsEmptyID(t *testing.T) {
	s := newTestStore(t)
	err := s.UpsertLot(Lot{Title: "Monitor sem id"})
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func idsOf(lots []Lot) []string {
	ids := make([]string, len(lots))
	for i, lot := range lots {
		ids[i] = lot.ID
	}
	return ids
}
