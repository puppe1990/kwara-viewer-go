package handlers

import (
	"context"
	"fmt"
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

type stubDetailer struct {
	stubSearcher
	details map[string]store.Lot
	called  []string
}

func (s *stubDetailer) LotDetails(_ context.Context, id string) (store.Lot, error) {
	s.called = append(s.called, id)
	lot, ok := s.details[id]
	if !ok {
		return store.Lot{}, fmt.Errorf("lote %s não encontrado", id)
	}
	return lot, nil
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
		PickupCity:      "Taboão da Serra",
		PickupAddress:   "Jardim Triangulo - Taboão da Serra/SP",
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
	if !strings.Contains(body, "Taboão da Serra") {
		t.Fatal("missing pickup city")
	}
	if !strings.Contains(body, "Jardim Triangulo - Taboão da Serra/SP") {
		t.Fatal("missing pickup address")
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

func TestCatalogSync_FillsPickupFromLotDetails(t *testing.T) {
	incoming := store.Lot{
		ID:          "1746417152014944098",
		Title:       "Monitor gamer",
		KwaraID:     "K-3091",
		ListingID:   "1742065509714625998",
		PickupState: "SP",
		EndsAt:      "2026-12-05T12:00:00Z",
	}
	searcher := &stubDetailer{
		stubSearcher: stubSearcher{items: []store.Lot{incoming}},
		details: map[string]store.Lot{
			"1746417152014944098": {
				ID:            "1746417152014944098",
				PickupState:   "SP",
				PickupCity:    "Embu das Artes",
				PickupAddress: "Jardim Eliza - Embu das Artes/SP",
			},
		},
	}
	h, s := newCatalog(t, searcher)

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(url.Values{
		"q":            {"monitor"},
		"pickup_state": {"SP"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Sync(rr, req)

	got, err := s.ListLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("stored = %+v", got)
	}
	if got[0].PickupCity != "Embu das Artes" {
		t.Errorf("city = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Eliza - Embu das Artes/SP" {
		t.Errorf("address = %q", got[0].PickupAddress)
	}
	if len(searcher.called) != 1 || searcher.called[0] != incoming.ID {
		t.Errorf("details called = %v", searcher.called)
	}
}

func TestCatalogSync_FillsPickupOncePerListing(t *testing.T) {
	listing := "1742065509714625998"
	first := store.Lot{
		ID: "a", Title: "Monitor A", ListingID: listing, KwaraID: "K-3091",
		PickupState: "SP", EndsAt: "2026-12-05T12:00:00Z",
	}
	second := store.Lot{
		ID: "b", Title: "Monitor B", ListingID: listing, KwaraID: "K-3091",
		PickupState: "SP", EndsAt: "2026-12-05T12:00:00Z",
	}
	searcher := &stubDetailer{
		stubSearcher: stubSearcher{items: []store.Lot{first, second}},
		details: map[string]store.Lot{
			"a": {
				ID:            "a",
				PickupState:   "SP",
				PickupCity:    "Embu das Artes",
				PickupAddress: "Jardim Eliza - Embu das Artes/SP",
			},
		},
	}
	h, s := newCatalog(t, searcher)

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(url.Values{
		"q":            {"monitor"},
		"pickup_state": {"SP"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Sync(httptest.NewRecorder(), req)

	got, err := s.ListLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("stored = %+v", got)
	}
	for _, lot := range got {
		if lot.PickupCity != "Embu das Artes" {
			t.Errorf("%s city = %q", lot.ID, lot.PickupCity)
		}
		if lot.PickupAddress != "Jardim Eliza - Embu das Artes/SP" {
			t.Errorf("%s address = %q", lot.ID, lot.PickupAddress)
		}
	}
	if len(searcher.called) != 1 || searcher.called[0] != "a" {
		t.Errorf("details called = %v, want one fetch per listing", searcher.called)
	}
}

func TestEnrichStoredPickup_PatchesMissingCity(t *testing.T) {
	s := setupTestStore(t)
	if err := s.UpsertLot(store.Lot{
		ID: "1", Title: "Monitor Dell", PickupState: "SP", EndsAt: "2026-12-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	d := &stubDetailer{
		details: map[string]store.Lot{
			"1": {
				ID:            "1",
				PickupCity:    "Taboão da Serra",
				PickupAddress: "Jardim Triangulo - Taboão da Serra/SP",
				PickupState:   "SP",
			},
		},
	}
	if err := EnrichStoredPickup(context.Background(), s, d); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListLots(store.LotFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].PickupCity != "Taboão da Serra" {
		t.Errorf("city = %q", got[0].PickupCity)
	}
	if got[0].PickupAddress != "Jardim Triangulo - Taboão da Serra/SP" {
		t.Errorf("address = %q", got[0].PickupAddress)
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

func TestCatalogHome_OpenGraphCard(t *testing.T) {
	h, s := newCatalog(t, &stubSearcher{})
	seedDellSP(t, s)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.Home(rr, req)
	body := rr.Body.String()

	want := []string{
		`property="og:title" content="Prancheta Kwara"`,
		`property="og:description" content="Prancheta de lotes da Kwara gravados no SQLite."`,
		`property="og:image" content="https://cais.example.com/static/og.png"`,
		`property="og:image:width" content="1200"`,
		`property="og:image:height" content="630"`,
		`property="og:image:alt" content="Prancheta Kwara — lotes no SQLite"`,
		`property="og:url" content="https://cais.example.com/"`,
		`name="twitter:card" content="summary_large_image"`,
		`name="twitter:image" content="https://cais.example.com/static/og.png"`,
		`name="twitter:title" content="Prancheta Kwara"`,
	}
	for _, snippet := range want {
		if !strings.Contains(body, snippet) {
			t.Errorf("missing %q", snippet)
		}
	}
}
