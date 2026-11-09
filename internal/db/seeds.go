package db

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/puppe1990/kwara-viewer/internal/kwara"
	"github.com/puppe1990/kwara-viewer/internal/store"
)

//go:embed seed-lots.json
var seedLotsJSON []byte

// Known pickup places from Kwara lot details (listing.location + pickUpAddress).
var seedPickupByKwara = map[string][2]string{
	"K-3087": {"Taboão da Serra", "Jardim Triangulo - Taboão da Serra/SP"},
	"K-3091": {"Embu das Artes", "Jardim Eliza - Embu das Artes/SP"},
}

// RunSeeds upserts the bundled SP monitor dump when the table is empty,
// then fills blank pickup city/address from the seed snapshot.
func RunSeeds(s store.Store) error {
	rows, err := loadSeedRows()
	if err != nil {
		return err
	}
	n, err := s.CountLots(store.LotFilter{})
	if err != nil {
		return fmt.Errorf("count lots before seed: %w", err)
	}
	if n == 0 {
		for _, row := range rows {
			lot, err := kwara.LotFromSeedRow(row)
			if err != nil {
				return fmt.Errorf("seed row: %w", err)
			}
			if !kwara.IsMonitorLot(lot.Title) {
				continue
			}
			lot = applySeedPickup(lot)
			if err := s.UpsertLot(lot); err != nil {
				return err
			}
		}
	}
	return backfillPickupFromSeed(s, rows)
}

func loadSeedRows() ([]map[string]any, error) {
	var rows []map[string]any
	if err := json.Unmarshal(seedLotsJSON, &rows); err != nil {
		return nil, fmt.Errorf("seed json: %w", err)
	}
	return rows, nil
}

func applySeedPickup(lot store.Lot) store.Lot {
	place, ok := seedPickupByKwara[lot.KwaraID]
	if !ok {
		return lot
	}
	if lot.PickupCity == "" {
		lot.PickupCity = place[0]
	}
	if lot.PickupAddress == "" {
		lot.PickupAddress = place[1]
	}
	return lot
}

func backfillPickupFromSeed(s store.Store, rows []map[string]any) error {
	for _, row := range rows {
		lot, err := kwara.LotFromSeedRow(row)
		if err != nil {
			continue
		}
		lot = applySeedPickup(lot)
		if lot.PickupCity == "" && lot.PickupAddress == "" {
			continue
		}
		if err := s.PatchPickup(lot.ID, lot.PickupCity, lot.PickupAddress, lot.PickupState); err != nil {
			return err
		}
	}
	return nil
}
