package handlers

import (
	"time"

	"github.com/puppe1990/kwara-viewer/internal/kwara"
	"github.com/puppe1990/kwara-viewer/internal/store"
)

var brt = time.FixedZone("BRT", -3*60*60)

type lotView struct {
	store.Lot
	BidLabel  string
	EndsLabel string
	Open      bool
}

func newLotViews(lots []store.Lot) []lotView {
	now := time.Now().UTC()
	views := make([]lotView, 0, len(lots))
	for _, lot := range lots {
		views = append(views, newLotView(lot, now))
	}
	return views
}

func newLotView(lot store.Lot, now time.Time) lotView {
	ends, open := lotTiming(lot, now)
	bid := "—"
	if lot.CurrentBidCents > 0 {
		bid = kwara.FormatReais(lot.CurrentBidCents)
	}
	return lotView{Lot: lot, BidLabel: bid, EndsLabel: ends, Open: open}
}

func lotTiming(lot store.Lot, now time.Time) (string, bool) {
	if lot.EndsAt == "" {
		return "sem prazo", !lot.Ended
	}
	ends, err := time.Parse(time.RFC3339Nano, lot.EndsAt)
	if err != nil {
		ends, err = time.Parse(time.RFC3339, lot.EndsAt)
	}
	if err != nil {
		return lot.EndsAt, !lot.Ended
	}
	open := !lot.Ended && ends.After(now)
	return ends.In(brt).Format("02/01 15h04") + " BRT", open
}
