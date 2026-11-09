package kwara

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/puppe1990/kwara-viewer/internal/store"
)

const RSCParseError = "não deu para ler a busca do site"

const buscaRouterStateTree = "%5B%22%22%2C%7B%22children%22%3A%5B%22busca%22%2C%7B%22children%22%3A%5B%22__PAGE__%22%2C%7B%7D%2Cnull%2C%22refetch%22%2C0%5D%7D%2Cnull%2Cnull%2C0%5D%7D%2Cnull%2Cnull%2C16%5D"

type SearchPage struct {
	Items []store.Lot
	Total int
}

func ParseRscSearch(body string) (SearchPage, error) {
	key := `"searchResult":`
	idx := strings.Index(body, key)
	if idx < 0 {
		return SearchPage{}, fmt.Errorf("%s", RSCParseError)
	}
	start := strings.Index(body[idx+len(key):], "{")
	if start < 0 {
		return SearchPage{}, fmt.Errorf("%s", RSCParseError)
	}
	start += idx + len(key)
	raw := balancedObject(body, start)
	if raw == "" {
		return SearchPage{}, fmt.Errorf("%s", RSCParseError)
	}
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return SearchPage{}, fmt.Errorf("%s", RSCParseError)
	}
	return parseSearchPayload(payload)
}

func parseSearchPayload(payload any) (SearchPage, error) {
	root, _ := payload.(map[string]any)
	itemsRaw := itemsFrom(payload)
	total := totalFrom(root)
	items := make([]store.Lot, 0, len(itemsRaw))
	for _, row := range itemsRaw {
		lot, err := CompactLot(row)
		if err != nil {
			continue
		}
		items = append(items, lot)
	}
	return SearchPage{Items: items, Total: total}, nil
}

func itemsFrom(payload any) []any {
	if arr, ok := payload.([]any); ok {
		return arr
	}
	root, _ := payload.(map[string]any)
	if root == nil {
		return nil
	}
	if arr, ok := root["items"].([]any); ok {
		return arr
	}
	if data, ok := root["data"].(map[string]any); ok {
		if arr, ok := data["items"].([]any); ok {
			return arr
		}
	}
	return nil
}

func totalFrom(root map[string]any) int {
	if root == nil {
		return 0
	}
	if n := intField(root, "totalItems"); n > 0 {
		return int(n)
	}
	if n := intField(root, "total"); n > 0 {
		return int(n)
	}
	if data := mapField(root, "data"); len(data) > 0 {
		if n := intField(data, "totalItems"); n > 0 {
			return int(n)
		}
	}
	return 0
}

func balancedObject(source string, start int) string {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(source); i++ {
		ch := source[i]
		if inString {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : i+1]
			}
		}
	}
	return ""
}

func clampTake(take int) int {
	if take <= 0 {
		return 20
	}
	if take > 50 {
		return 50
	}
	return take
}
