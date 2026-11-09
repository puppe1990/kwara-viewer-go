package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/kwara-viewer/internal/kwara"
	"github.com/puppe1990/kwara-viewer/internal/store"
)

type LotSearcher interface {
	SearchAll(ctx context.Context, q string, take int) ([]store.Lot, error)
}

type LotDetailer interface {
	LotDetails(ctx context.Context, id string) (store.Lot, error)
}

type CatalogHandler struct {
	views    *view.Renderer
	store    store.Store
	searcher LotSearcher
	site     meta.Site
	catalog  *i18n.Catalog
	cfg      cais.Config
}

func NewCatalogHandler(views *view.Renderer, s store.Store, searcher LotSearcher, site meta.Site, catalog *i18n.Catalog, cfg cais.Config) *CatalogHandler {
	return &CatalogHandler{views: views, store: s, searcher: searcher, site: site, catalog: catalog, cfg: cfg}
}

func (h *CatalogHandler) t(r *http.Request, key string, args ...any) string {
	return i18n.CatalogOr(r, h.catalog).T(key, args...)
}

func (h *CatalogHandler) Home(w http.ResponseWriter, r *http.Request) {
	q := parseCatalogQuery(r)
	filter := q.filter()
	matched, err := h.store.CountLots(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pager := q.pager(matched)
	filter.Limit = catalogPageSize
	filter.Offset = (pager.Page - 1) * catalogPageSize
	lots, err := h.store.ListLots(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	total, err := h.store.CountLots(store.LotFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.write(w, r, "home", h.t(r, "home.title"), map[string]any{
		"Query":    q.Query,
		"Pickup":   q.Pickup,
		"OpenOnly": q.OpenOnly,
		"Sort":     q.Sort,
		"Desc":     q.Desc,
		"Lots":     newLotViews(lots),
		"Shown":    len(lots),
		"Matched":  matched,
		"Stored":   total,
		"Headers":  q.headers(h.sortLabels(r)),
		"Pager":    pager,
	})
}

func (h *CatalogHandler) sortLabels(r *http.Request) [][2]string {
	return [][2]string{
		{"kwara", h.t(r, "home.lot")},
		{"title", h.t(r, "home.title_col")},
		{"city", h.t(r, "home.pickup")},
		{"bid", h.t(r, "home.bid")},
		{"ends", h.t(r, "home.ends")},
	}
}

func (h *CatalogHandler) Sync(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q := strings.TrimSpace(r.FormValue("q"))
	if q == "" {
		q = "monitor"
	}
	pickup := strings.TrimSpace(r.FormValue("pickup_state"))
	saved, err := h.pullLots(r.Context(), q, pickup)
	if err != nil {
		flash.Set(w, "alert", err.Error(), h.cfg.CookieSecure())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	flash.Set(w, "notice", fmt.Sprintf("Gravados %d lotes no SQLite.", saved), h.cfg.CookieSecure())
	http.Redirect(w, r, "/?q="+q+"&pickup="+pickup, http.StatusSeeOther)
}

func (h *CatalogHandler) pullLots(ctx context.Context, q, pickup string) (int, error) {
	if h.searcher == nil {
		return 0, fmt.Errorf("busca da Kwara não configurada")
	}
	items, err := h.searcher.SearchAll(ctx, q, 20)
	if err != nil {
		return 0, err
	}
	if d, ok := h.searcher.(LotDetailer); ok {
		items = fillPickupPlaces(ctx, d, items)
	}
	saved := 0
	for _, lot := range items {
		if pickup != "" && !strings.EqualFold(lot.PickupState, pickup) {
			continue
		}
		if strings.EqualFold(q, "monitor") && !kwara.IsMonitorLot(lot.Title) {
			continue
		}
		if err := h.store.UpsertLot(lot); err != nil {
			return saved, err
		}
		saved++
	}
	return saved, nil
}

func fillPickupPlaces(ctx context.Context, d LotDetailer, lots []store.Lot) []store.Lot {
	cache := map[string]store.Lot{}
	for i, lot := range lots {
		if lot.PickupCity != "" && lot.PickupAddress != "" {
			continue
		}
		key := pickupGroupKey(lot)
		detail, ok := cache[key]
		if !ok {
			fetched, err := d.LotDetails(ctx, lot.ID)
			if err != nil {
				continue
			}
			detail = fetched
			cache[key] = detail
		}
		if lots[i].PickupCity == "" {
			lots[i].PickupCity = detail.PickupCity
		}
		if lots[i].PickupAddress == "" {
			lots[i].PickupAddress = detail.PickupAddress
		}
		if lots[i].PickupState == "" {
			lots[i].PickupState = detail.PickupState
		}
	}
	return lots
}

func pickupGroupKey(lot store.Lot) string {
	if lot.ListingID != "" {
		return "listing:" + lot.ListingID
	}
	if lot.KwaraID != "" {
		return "kwara:" + lot.KwaraID
	}
	return "lot:" + lot.ID
}

func EnrichStoredPickup(ctx context.Context, s store.Store, d LotDetailer) error {
	lots, err := s.ListLots(store.LotFilter{})
	if err != nil {
		return err
	}
	var missing []store.Lot
	for _, lot := range lots {
		if lot.PickupCity == "" || lot.PickupAddress == "" {
			missing = append(missing, lot)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	for _, lot := range fillPickupPlaces(ctx, d, missing) {
		if lot.PickupCity == "" && lot.PickupAddress == "" {
			continue
		}
		if err := s.PatchPickup(lot.ID, lot.PickupCity, lot.PickupAddress, lot.PickupState); err != nil {
			return err
		}
	}
	return nil
}

func (h *CatalogHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
}

func (h *CatalogHandler) write(w http.ResponseWriter, r *http.Request, name, title string, extra map[string]any) {
	extra["Title"] = title
	extra["ActiveNav"] = "home"
	view.Write(w, r, h.views, view.Page{
		Layout: "app",
		Name:   name,
		Data:   amarraData(r, h.site, extra),
	}, h.cfg)
}
