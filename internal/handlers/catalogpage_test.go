package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCatalogQuery_ReadsPageSortAndOpen(t *testing.T) {
	req := httptest.NewRequest("GET", "/?q=Dell&pickup=SP&city=Embu+das+Artes&bairro=Jardim+Eliza&open=0&sort=bid&dir=desc&page=3", nil)
	q := parseCatalogQuery(req)
	if q.Query != "Dell" || q.Pickup != "SP" || q.City != "Embu das Artes" || q.Bairro != "Jardim Eliza" || q.OpenOnly || q.Sort != "bid" || !q.Desc || q.Page != 3 {
		t.Fatalf("%+v", q)
	}
}

func TestCatalogQuery_HrefKeepsFilterSortAndPage(t *testing.T) {
	q := catalogQuery{Query: "Dell", City: "Embu das Artes", Bairro: "Jardim Eliza", OpenOnly: false, Sort: "bid", Desc: true}
	got := q.href(2, q.Sort, q.Desc)
	for _, part := range []string{"q=Dell", "city=Embu", "bairro=Jardim", "open=0", "sort=bid", "dir=desc", "page=2"} {
		if !strings.Contains(got, part) {
			t.Errorf("%s missing in %q", part, got)
		}
	}
}

func TestCatalogQuery_PagerClampsAndLinks(t *testing.T) {
	q := catalogQuery{Page: 99, Sort: "city", OpenOnly: true}
	p := q.pager(11)
	if p.Page != 2 || p.Pages != 2 {
		t.Fatalf("pager = %+v", p)
	}
	if !p.HasPrev || p.HasNext {
		t.Fatalf("nav flags %+v", p)
	}
	if strings.Contains(p.Prev, "page=") {
		t.Errorf("page 1 should omit page= in %q", p.Prev)
	}
	if !strings.Contains(p.Prev, "sort=city") {
		t.Errorf("prev = %q", p.Prev)
	}
	if len(p.Links) != 2 || !p.Links[1].Current {
		t.Fatalf("links = %+v", p.Links)
	}
}
