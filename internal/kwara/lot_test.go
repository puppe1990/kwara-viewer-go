package kwara

import (
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais/fakedata"
)

func TestCompactLot_MapsSearchItem(t *testing.T) {
	fakedata.Seed(21)
	title := "Monitor Dell " + fakedata.Title()
	raw := map[string]any{
		"id":                     "1739237630056335265",
		"number":                 "020",
		"title":                  title,
		"slug":                   "monitor-dell-teclado-e-mouse-1739237630056335265",
		"listingId":              "listing-1",
		"cachedPriceAmountCents": float64(18000),
		"cachedBidsCount":        float64(1),
		"currentEndAt":           "2026-11-09T17:04:45.000+00:00",
		"pickUpState":            "SP",
		"isEnded":                false,
		"listing": map[string]any{
			"id":      "listing-1",
			"title":   "Informática Ecobraz",
			"kwaraId": "K-3126",
		},
		"lotAuctionSettings": []any{
			map[string]any{"isCurrent": true, "startingBidCents": float64(10000)},
		},
	}

	lot, err := CompactLot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if lot.ID != "1739237630056335265" {
		t.Errorf("id = %q", lot.ID)
	}
	if lot.Title != title {
		t.Errorf("title = %q", lot.Title)
	}
	if lot.KwaraID != "K-3126" {
		t.Errorf("kwaraId = %q", lot.KwaraID)
	}
	if lot.CurrentBidCents != 18000 {
		t.Errorf("bid = %d", lot.CurrentBidCents)
	}
	if lot.StartingBidCents != 10000 {
		t.Errorf("start = %d", lot.StartingBidCents)
	}
	if lot.PickupState != "SP" {
		t.Errorf("pickup = %q", lot.PickupState)
	}
	if lot.Ended {
		t.Error("ended")
	}
	wantURL := "https://www.kwara.com.br/bens/monitor-dell-teclado-e-mouse-1739237630056335265"
	if lot.URL != wantURL {
		t.Errorf("url = %q, want %q", lot.URL, wantURL)
	}
}

func TestCompactLot_RequiresID(t *testing.T) {
	_, err := CompactLot(map[string]any{"title": "Monitor"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatReais(t *testing.T) {
	if got := FormatReais(18000); got != "R$ 180,00" {
		t.Errorf("got %q", got)
	}
	if got := FormatReais(123456); got != "R$ 1.234,56" {
		t.Errorf("got %q", got)
	}
}

func TestParseReais(t *testing.T) {
	cents, err := ParseReais("R$ 180,00")
	if err != nil || cents != 18000 {
		t.Fatalf("got %d %v", cents, err)
	}
	cents, err = ParseReais("R$ 1.234,56")
	if err != nil || cents != 123456 {
		t.Fatalf("got %d %v", cents, err)
	}
}

func TestIsMonitorLot(t *testing.T) {
	if !IsMonitorLot("Monitor Dell, teclado e mouse") {
		t.Fatal("dell bundle should count")
	}
	if IsMonitorLot("Notebook Dell") {
		t.Fatal("notebook should not count")
	}
	if IsMonitorLot("Suporte para monitor articulado") {
		t.Fatal("suporte should not count")
	}
	if IsMonitorLot("Base de monitor Dell") {
		t.Fatal("base should not count")
	}
}

func TestLotFromSeedRow_FillsIDFromURL(t *testing.T) {
	lot, err := LotFromSeedRow(map[string]any{
		"number":  "020",
		"kwaraId": "K-3126",
		"title":   "Monitor Dell, teclado e mouse",
		"bid":     "R$ 180,00",
		"bids":    float64(1),
		"endsAt":  "2026-11-09T17:04:45.000+00:00",
		"url":     "https://www.kwara.com.br/bens/monitor-dell-teclado-e-mouse-1739237630056335265",
	})
	if err != nil {
		t.Fatal(err)
	}
	if lot.ID != "1739237630056335265" {
		t.Errorf("id = %q", lot.ID)
	}
	if lot.Slug != "monitor-dell-teclado-e-mouse-1739237630056335265" {
		t.Errorf("slug = %q", lot.Slug)
	}
	if lot.CurrentBidCents != 18000 {
		t.Errorf("bid = %d", lot.CurrentBidCents)
	}
	if lot.PickupState != "SP" {
		t.Errorf("pickup default = %q", lot.PickupState)
	}
}
