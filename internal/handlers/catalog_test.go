package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/fakedata"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"

	appi18n "github.com/puppe1990/kwara-viewer/internal/i18n"
	"github.com/puppe1990/kwara-viewer/internal/store"
)

type stubSearcher struct {
	items []store.Lot
	err   error
	q     string
}

func (s *stubSearcher) SearchAll(_ context.Context, q string, _ int) ([]store.Lot, error) {
	s.q = q
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func newCatalog(t *testing.T, searcher LotSearcher) (*CatalogHandler, store.Store) {
	t.Helper()
	st := setupTestStore(t)
	h := NewCatalogHandler(setupTestViews(t), st, searcher, testSite(), appi18n.NewCatalog("pt"), cais.Config{})
	return h, st
}

func seedDellSP(t *testing.T, s store.Store) store.Lot {
	t.Helper()
	fakedata.Seed(42)
	lot := store.Lot{
		ID:              "1739",
		Number:          "020",
		Title:           "Monitor Dell P2219H " + fakedata.Title(),
		Slug:            "monitor-dell-1739",
		KwaraID:         "K-3126",
		CurrentBidCents: 18000,
		BidsCount:       1,
		EndsAt:          "2026-12-01T15:00:00Z",
		PickupState:     "SP",
		URL:             "https://www.kwara.com.br/bens/monitor-dell-1739",
	}
	if err := s.UpsertLot(lot); err != nil {
		t.Fatal(err)
	}
	return lot
}

func TestCatalogHome_ListsSeededLot(t *testing.T) {
	h, s := newCatalog(t, &stubSearcher{})
	lot := seedDellSP(t, s)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.Home(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="amarra-main"`) {
		t.Fatal("missing #amarra-main")
	}
	if !strings.Contains(body, lot.Title) {
		t.Fatalf("missing title %q in %s", lot.Title, body)
	}
	if !strings.Contains(body, "K-3126") {
		t.Fatal("missing kwara id")
	}
	if !strings.Contains(body, "R$ 180,00") {
		t.Fatal("missing bid")
	}
	if strings.Contains(body, "is running") {
		t.Fatal("scaffold heading still present")
	}
}

func TestCatalogHome_FiltersByQuery(t *testing.T) {
	h, s := newCatalog(t, &stubSearcher{})
	seedDellSP(t, s)
	other := store.Lot{ID: "2", Title: "Monitor LG 24", PickupState: "SP", EndsAt: "2026-12-01T15:00:00Z"}
	if err := s.UpsertLot(other); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?q=Dell", nil)
	rr := httptest.NewRecorder()
	h.Home(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "Monitor Dell") {
		t.Fatal("dell missing")
	}
	if strings.Contains(body, "Monitor LG 24") {
		t.Fatal("LG should be filtered out")
	}
}

func TestCatalogSync_UpsertsSearcherLots(t *testing.T) {
	fakedata.Seed(5)
	incoming := store.Lot{
		ID:              "88",
		Title:           "Monitor AOC " + fakedata.Title(),
		KwaraID:         "K-3091",
		CurrentBidCents: 5000,
		PickupState:     "SP",
		EndsAt:          "2026-12-05T12:00:00Z",
		URL:             "https://www.kwara.com.br/bens/monitor-aoc-88",
	}
	searcher := &stubSearcher{items: []store.Lot{incoming}}
	h, s := newCatalog(t, searcher)

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(url.Values{
		"q":            {"monitor"},
		"pickup_state": {"SP"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Sync(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if searcher.q != "monitor" {
		t.Errorf("search q = %q", searcher.q)
	}
	got, err := s.ListLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "88" {
		t.Fatalf("stored = %+v", got)
	}
	flashReq := &http.Request{Header: make(http.Header)}
	for _, c := range rr.Result().Cookies() {
		flashReq.AddCookie(c)
	}
	msg, ok := flash.Consume(flashReq)
	if !ok || !strings.Contains(msg.Message, "1") {
		t.Fatalf("flash = %+v ok=%v", msg, ok)
	}
}

func TestCatalogRobots_DisallowAll(t *testing.T) {
	h, _ := newCatalog(t, &stubSearcher{})
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	h.Robots(rr, req)
	if !strings.Contains(rr.Body.String(), "Disallow: /") {
		t.Errorf("robots = %q", rr.Body.String())
	}
}
