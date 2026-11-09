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

// RunSeeds upserts the bundled SP monitor dump when the table is empty.
func RunSeeds(s store.Store) error {
	n, err := s.CountLots(store.LotFilter{})
	if err != nil {
		return fmt.Errorf("count lots before seed: %w", err)
	}
	if n > 0 {
		return nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(seedLotsJSON, &rows); err != nil {
		return fmt.Errorf("seed json: %w", err)
	}
	for _, row := range rows {
		lot, err := kwara.LotFromSeedRow(row)
		if err != nil {
			return fmt.Errorf("seed row: %w", err)
		}
		if !kwara.IsMonitorLot(lot.Title) {
			continue
		}
		if err := s.UpsertLot(lot); err != nil {
			return err
		}
	}
	return nil
}
