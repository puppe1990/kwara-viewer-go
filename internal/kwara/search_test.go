package kwara

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestParseRscSearch_ExtractsSearchResult(t *testing.T) {
	body := `0:{"searchResult":{"items":[{"id":"1739","number":"020","title":"Monitor Dell","slug":"monitor-dell-1739","cachedPriceAmountCents":18000,"cachedBidsCount":1,"currentEndAt":"2026-11-09T17:04:45.000+00:00","pickUpState":"SP","isEnded":false,"listing":{"kwaraId":"K-3126"}}],"totalItems":1}}`
	page, err := ParseRscSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Errorf("total = %d", page.Total)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "1739" {
		t.Fatalf("items = %+v", page.Items)
	}
	if page.Items[0].KwaraID != "K-3126" {
		t.Errorf("kwaraId = %q", page.Items[0].KwaraID)
	}
}

func TestParseRscSearch_RejectsGarbage(t *testing.T) {
	_, err := ParseRscSearch("not a flight")
	if err == nil || !strings.Contains(err.Error(), "não deu para ler a busca do site") {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_SearchTextUsesBuscaRSC(t *testing.T) {
	var gotURL string
	var gotRSC string
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		gotRSC = req.Header.Get("rsc")
		body := `{"searchResult":{"items":[{"id":"1","title":"Monitor Dell","slug":"monitor-dell-1","pickUpState":"SP","listing":{"kwaraId":"K-3126"}}],"totalItems":1}}`
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/x-component"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})})

	page, err := client.Search(context.Background(), SearchQuery{Q: "monitor", Take: 20, Skip: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "https://www.kwara.com.br/busca?q=monitor") {
		t.Errorf("url = %q", gotURL)
	}
	if gotRSC != "1" {
		t.Errorf("rsc header = %q", gotRSC)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Monitor Dell" {
		t.Fatalf("page = %+v", page)
	}
}

func TestClient_SearchPaginatesUntilTotal(t *testing.T) {
	calls := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		page := "1"
		if strings.Contains(req.URL.RawQuery, "currentPage=2") {
			page = "2"
		}
		id := "1"
		if page == "2" {
			id = "2"
		}
		body := `{"searchResult":{"items":[{"id":"` + id + `","title":"Monitor ` + id + `","slug":"m-` + id + `","pickUpState":"SP"}],"totalItems":2}}`
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/x-component"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})})

	items, err := client.SearchAll(context.Background(), "monitor", 20)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}
