package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/puppe1990/kwara-viewer/internal/store"
)

const catalogPageSize = 10

type catalogQuery struct {
	Query    string
	Pickup   string
	City     string
	Bairro   string
	Sort     string
	OpenOnly bool
	Desc     bool
	Page     int
}

type sortHeader struct {
	Key    string
	Label  string
	Href   string
	Aria   string
	Active bool
}

type pageLink struct {
	N       int
	Href    string
	Current bool
}

type catalogPager struct {
	Page    int
	Pages   int
	Prev    string
	Next    string
	HasPrev bool
	HasNext bool
	Links   []pageLink
}

func parseCatalogQuery(r *http.Request) catalogQuery {
	q := catalogQuery{
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		Pickup:   strings.TrimSpace(r.URL.Query().Get("pickup")),
		City:     strings.TrimSpace(r.URL.Query().Get("city")),
		Bairro:   strings.TrimSpace(r.URL.Query().Get("bairro")),
		OpenOnly: r.URL.Query().Get("open") != "0",
		Page:     1,
	}
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		q.Page = n
	}
	q.Sort, q.Desc = normalizeLotSort(r.URL.Query().Get("sort"), r.URL.Query().Get("dir"))
	return q
}

func normalizeLotSort(sort, dir string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "kwara", "title", "city", "bid", "ends":
		return strings.ToLower(strings.TrimSpace(sort)), strings.EqualFold(strings.TrimSpace(dir), "desc")
	default:
		return "", false
	}
}

func (q catalogQuery) filter() store.LotFilter {
	return store.LotFilter{
		Query:        q.Query,
		PickupState:  q.Pickup,
		PickupCity:   q.City,
		Neighborhood: q.Bairro,
		OpenOnly:     q.OpenOnly,
		Sort:         q.Sort,
		Desc:         q.Desc,
	}
}

func (q catalogQuery) href(page int, sort string, desc bool) string {
	v := url.Values{}
	if q.Query != "" {
		v.Set("q", q.Query)
	}
	if q.Pickup != "" {
		v.Set("pickup", q.Pickup)
	}
	if q.City != "" {
		v.Set("city", q.City)
	}
	if q.Bairro != "" {
		v.Set("bairro", q.Bairro)
	}
	if !q.OpenOnly {
		v.Set("open", "0")
	}
	if sort != "" {
		v.Set("sort", sort)
		if desc {
			v.Set("dir", "desc")
		} else {
			v.Set("dir", "asc")
		}
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	enc := v.Encode()
	if enc == "" {
		return "/"
	}
	return "/?" + enc
}

func (q catalogQuery) headers(cols [][2]string) []sortHeader {
	out := make([]sortHeader, 0, len(cols))
	for _, col := range cols {
		key, label := col[0], col[1]
		active := q.Sort == key
		nextDesc := false
		if active {
			nextDesc = !q.Desc
		} else if key == "bid" {
			nextDesc = true
		}
		aria := "none"
		if active && q.Desc {
			aria = "descending"
		} else if active {
			aria = "ascending"
		}
		out = append(out, sortHeader{
			Key:    key,
			Label:  label,
			Href:   q.href(1, key, nextDesc),
			Aria:   aria,
			Active: active,
		})
	}
	return out
}

func (q catalogQuery) pager(matched int) catalogPager {
	pages := (matched + catalogPageSize - 1) / catalogPageSize
	if pages < 1 {
		pages = 1
	}
	page := q.Page
	if page > pages {
		page = pages
	}
	p := catalogPager{Page: page, Pages: pages}
	if pages == 1 {
		return p
	}
	p.HasPrev = page > 1
	p.HasNext = page < pages
	if p.HasPrev {
		p.Prev = q.href(page-1, q.Sort, q.Desc)
	}
	if p.HasNext {
		p.Next = q.href(page+1, q.Sort, q.Desc)
	}
	p.Links = make([]pageLink, 0, pages)
	for n := 1; n <= pages; n++ {
		p.Links = append(p.Links, pageLink{
			N:       n,
			Href:    q.href(n, q.Sort, q.Desc),
			Current: n == page,
		})
	}
	return p
}
