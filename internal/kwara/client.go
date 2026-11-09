package kwara

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/puppe1990/kwara-viewer/internal/store"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36"

type SearchQuery struct {
	Q    string
	Take int
	Skip int
}

type Client struct {
	http    *http.Client
	timeout time.Duration
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{http: httpClient, timeout: 15 * time.Second}
}

func (c *Client) LotDetails(ctx context.Context, id string) (store.Lot, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return store.Lot{}, fmt.Errorf("lot details: empty id")
	}
	u := APIOrigin + "/web/lots/" + url.PathEscape(id) + "/details?includeBids=true&isBot=false"
	payload, err := c.getJSON(ctx, u)
	if err != nil {
		return store.Lot{}, err
	}
	lot, err := CompactLot(payload)
	if err != nil {
		return store.Lot{}, fmt.Errorf("lot details %s: %w", id, err)
	}
	return lot, nil
}

func (c *Client) Search(ctx context.Context, q SearchQuery) (SearchPage, error) {
	limit := clampTake(q.Take)
	offset := q.Skip
	if offset < 0 {
		offset = 0
	}
	page := 1
	if offset > 0 {
		page = offset/limit + 1
	}
	u := SiteOrigin + "/busca?q=" + url.QueryEscape(strings.TrimSpace(q.Q))
	if offset > 0 {
		u += "&currentPage=" + fmt.Sprintf("%d", page)
	}
	body, err := c.getRSC(ctx, u)
	if err != nil {
		return SearchPage{}, err
	}
	return ParseRscSearch(body)
}

func (c *Client) SearchAll(ctx context.Context, q string, take int) ([]store.Lot, error) {
	limit := clampTake(take)
	var all []store.Lot
	skip := 0
	for page := 0; page < 40; page++ {
		result, err := c.Search(ctx, SearchQuery{Q: q, Take: limit, Skip: skip})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if len(result.Items) == 0 {
			break
		}
		if result.Total > 0 && len(all) >= result.Total {
			break
		}
		if result.Total == 0 && len(result.Items) < limit {
			break
		}
		skip += limit
	}
	return all, nil
}

func (c *Client) getJSON(ctx context.Context, rawURL string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("kwara request: %w", err)
	}
	req.Header.Set("Origin", SiteOrigin)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kwara fetch %s: %w", req.URL.Path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, req.URL.Path)
	}
	var payload any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("kwara json %s: %w", req.URL.Path, err)
	}
	return payload, nil
}

func (c *Client) getRSC(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("kwara request: %w", err)
	}
	req.Header.Set("Origin", SiteOrigin)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("rsc", "1")
	req.Header.Set("next-url", "/busca")
	req.Header.Set("next-router-state-tree", buscaRouterStateTree)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("kwara fetch %s: %w", req.URL.Path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("HTTP %d %s", resp.StatusCode, req.URL.Path)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("kwara body %s: %w", req.URL.Path, err)
	}
	return string(b), nil
}
