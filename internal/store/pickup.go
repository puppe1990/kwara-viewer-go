package store

import (
	"fmt"
	"sort"
	"strings"
)

type PickupFacets struct {
	Cities        []string
	Neighborhoods []string
}

func NeighborhoodFromAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if i := strings.Index(addr, " - "); i > 0 {
		return strings.TrimSpace(addr[:i])
	}
	return addr
}

func (s *SQLiteStore) PickupFacets(city string) (PickupFacets, error) {
	rows, err := s.db.Query(`SELECT pickup_city, pickup_address FROM lots`)
	if err != nil {
		return PickupFacets{}, fmt.Errorf("pickup facets: %w", err)
	}
	defer rows.Close()

	city = strings.TrimSpace(city)
	var cities, bairros []string
	for rows.Next() {
		var storedCity, addr string
		if err := rows.Scan(&storedCity, &addr); err != nil {
			return PickupFacets{}, fmt.Errorf("scan pickup facets: %w", err)
		}
		cities = append(cities, storedCity)
		if city == "" || strings.EqualFold(storedCity, city) {
			bairros = append(bairros, NeighborhoodFromAddress(addr))
		}
	}
	if err := rows.Err(); err != nil {
		return PickupFacets{}, fmt.Errorf("pickup facets: %w", err)
	}
	return PickupFacets{
		Cities:        uniqueSorted(cities),
		Neighborhoods: uniqueSorted(bairros),
	}, nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
