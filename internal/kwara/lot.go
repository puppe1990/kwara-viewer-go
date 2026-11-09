package kwara

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/puppe1990/kwara-viewer/internal/store"
)

const SiteOrigin = "https://www.kwara.com.br"
const APIOrigin = "https://api.kwara.com.br"

var trailingID = regexp.MustCompile(`-(\d+)$`)

func CompactLot(raw any) (store.Lot, error) {
	item := normalizeLot(raw)
	id := stringField(item, "id")
	if id == "" {
		id = stringField(item, "lotId")
	}
	if id == "" {
		return store.Lot{}, fmt.Errorf("compact lot: missing id")
	}
	listing := mapField(item, "listing")
	slug := stringField(item, "slug")
	listingID := stringField(item, "listingId")
	if listingID == "" {
		listingID = stringField(listing, "id")
	}
	setting := currentAuctionSetting(item)
	return store.Lot{
		ID:               id,
		Number:           stringField(item, "number"),
		Title:            stringField(item, "title"),
		Slug:             slug,
		ListingID:        listingID,
		ListingTitle:     stringField(listing, "title"),
		KwaraID:          stringField(listing, "kwaraId"),
		CurrentBidCents:  intField(item, "cachedPriceAmountCents"),
		StartingBidCents: intField(setting, "startingBidCents"),
		BidsCount:        int(intField(item, "cachedBidsCount")),
		EndsAt:           firstString(item, "currentEndAt", "scheduledEndAt"),
		PickupState:      stringField(item, "pickUpState"),
		Ended:            boolField(item, "isEnded"),
		URL:              lotURL(slug, id),
	}, nil
}

func LotFromSeedRow(raw map[string]any) (store.Lot, error) {
	href := stringField(raw, "url")
	id := stringField(raw, "id")
	slug := stringField(raw, "slug")
	if href != "" {
		path := href
		if u, err := url.Parse(href); err == nil {
			path = strings.Trim(u.Path, "/")
		}
		if i := strings.LastIndex(path, "/"); i >= 0 {
			path = path[i+1:]
		}
		if slug == "" {
			slug = path
		}
		if id == "" {
			if m := trailingID.FindStringSubmatch(path); len(m) == 2 {
				id = m[1]
			}
		}
	}
	if id == "" {
		return store.Lot{}, fmt.Errorf("seed lot: missing id (title %q)", stringField(raw, "title"))
	}
	cents, err := ParseReais(stringField(raw, "bid"))
	if err != nil {
		cents = intField(raw, "currentBidCents")
	}
	pickup := stringField(raw, "pickupState")
	if pickup == "" {
		pickup = "SP"
	}
	return store.Lot{
		ID:              id,
		Number:          stringField(raw, "number"),
		Title:           stringField(raw, "title"),
		Slug:            slug,
		KwaraID:         firstString(raw, "kwaraId", "kwara_id"),
		CurrentBidCents: cents,
		BidsCount:       int(intField(raw, "bids")),
		EndsAt:          stringField(raw, "endsAt"),
		PickupState:     pickup,
		Ended:           boolField(raw, "ended"),
		URL:             lotURL(slug, id),
	}, nil
}

func FormatReais(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	whole := cents / 100
	frac := cents % 100
	digits := strconv.FormatInt(whole, 10)
	var grouped strings.Builder
	n := len(digits)
	for i, ch := range digits {
		if i > 0 && (n-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(ch)
	}
	out := fmt.Sprintf("R$ %s,%02d", grouped.String(), frac)
	if neg {
		return "-" + out
	}
	return out
}

func ParseReais(s string) (int64, error) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(s, "R$", ""))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if cleaned == "" {
		return 0, fmt.Errorf("parse reais: empty %q", s)
	}
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("parse reais %q: %w", s, err)
	}
	return int64(f*100 + 0.5), nil
}

func IsMonitorLot(title string) bool {
	t := strings.ToLower(title)
	if !strings.Contains(t, "monitor") {
		return false
	}
	if strings.Contains(t, "suporte") || strings.Contains(t, "base de monitor") {
		return false
	}
	return true
}

func lotURL(slug, id string) string {
	if slug == "" && id == "" {
		return ""
	}
	if slug != "" {
		return SiteOrigin + "/bens/" + slug
	}
	return SiteOrigin + "/bens/" + id
}

func currentAuctionSetting(item map[string]any) map[string]any {
	if cur := mapField(item, "currentLotAuctionSetting"); len(cur) > 0 {
		return cur
	}
	settings, _ := item["lotAuctionSettings"].([]any)
	for _, row := range settings {
		m, _ := row.(map[string]any)
		if boolField(m, "isCurrent") {
			return m
		}
	}
	if len(settings) > 0 {
		m, _ := settings[0].(map[string]any)
		return m
	}
	return map[string]any{}
}

func normalizeLot(raw any) map[string]any {
	item, _ := raw.(map[string]any)
	if item == nil {
		return map[string]any{}
	}
	merged := cloneMap(item)
	nestedItem := mapField(item, "item")
	nestedLot := mapField(item, "lot")
	for k, v := range nestedItem {
		merged[k] = v
	}
	for k, v := range nestedLot {
		merged[k] = v
	}
	if stringField(merged, "id") == "" {
		if v := stringField(item, "lotId"); v != "" {
			merged["id"] = v
		}
	}
	return merged
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mapField(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	if v == nil {
		return map[string]any{}
	}
	return v
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := stringField(m, key); s != "" {
			return s
		}
	}
	return ""
}

func intField(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

func boolField(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}
