package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Lot struct {
	ID               string
	Number           string
	Title            string
	Slug             string
	ListingID        string
	ListingTitle     string
	KwaraID          string
	CurrentBidCents  int64
	StartingBidCents int64
	BidsCount        int
	EndsAt           string
	PickupState      string
	PickupCity       string
	PickupAddress    string
	Ended            bool
	URL              string
	SyncedAt         time.Time
}

type LotFilter struct {
	Query       string
	PickupState string
	OpenOnly    bool
	Now         time.Time
	Sort        string
	Desc        bool
	Limit       int
	Offset      int
}

func (s *SQLiteStore) UpsertLot(lot Lot) error {
	if strings.TrimSpace(lot.ID) == "" {
		return fmt.Errorf("upsert lot: empty id (title %q)", lot.Title)
	}
	lot.EndsAt = normalizeEndsAt(lot.EndsAt)
	ended := 0
	if lot.Ended {
		ended = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO lots (
			id, number, title, slug, listing_id, listing_title, kwara_id,
			current_bid_cents, starting_bid_cents, bids_count, ends_at,
			pickup_state, pickup_city, pickup_address, ended, url, synced_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			number = excluded.number,
			title = excluded.title,
			slug = excluded.slug,
			listing_id = excluded.listing_id,
			listing_title = excluded.listing_title,
			kwara_id = excluded.kwara_id,
			current_bid_cents = excluded.current_bid_cents,
			starting_bid_cents = excluded.starting_bid_cents,
			bids_count = excluded.bids_count,
			ends_at = excluded.ends_at,
			pickup_state = excluded.pickup_state,
			pickup_city = excluded.pickup_city,
			pickup_address = excluded.pickup_address,
			ended = excluded.ended,
			url = excluded.url,
			synced_at = datetime('now'),
			updated_at = datetime('now')
	`, lot.ID, lot.Number, lot.Title, lot.Slug, lot.ListingID, lot.ListingTitle, lot.KwaraID,
		lot.CurrentBidCents, lot.StartingBidCents, lot.BidsCount, lot.EndsAt,
		lot.PickupState, lot.PickupCity, lot.PickupAddress, ended, lot.URL)
	if err != nil {
		return fmt.Errorf("upsert lot %s: %w", lot.ID, err)
	}
	return nil
}

func (s *SQLiteStore) ListLots(filter LotFilter) ([]Lot, error) {
	now := filter.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	where, args := lotWhere(filter, now)
	order, orderArgs := lotOrder(filter, now)
	query := `
		SELECT id, number, title, slug, listing_id, listing_title, kwara_id,
			current_bid_cents, starting_bid_cents, bids_count, ends_at,
			pickup_state, pickup_city, pickup_address, ended, url, synced_at
		FROM lots
		WHERE ` + where + `
		` + order
	args = append(args, orderArgs...)
	if filter.Limit > 0 {
		query += " LIMIT ? OFFSET ?"
		offset := filter.Offset
		if offset < 0 {
			offset = 0
		}
		args = append(args, filter.Limit, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()
	return scanLots(rows)
}

func (s *SQLiteStore) CountLots(filter LotFilter) (int, error) {
	now := filter.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	where, args := lotWhere(filter, now)
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM lots WHERE `+where, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count lots: %w", err)
	}
	return n, nil
}

func (s *SQLiteStore) PatchPickup(id, city, address, state string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("patch pickup: empty id")
	}
	_, err := s.db.Exec(`
		UPDATE lots SET
			pickup_city = CASE WHEN pickup_city = '' THEN ? ELSE pickup_city END,
			pickup_address = CASE WHEN pickup_address = '' THEN ? ELSE pickup_address END,
			pickup_state = CASE WHEN pickup_state = '' THEN ? ELSE pickup_state END,
			updated_at = datetime('now')
		WHERE id = ?
	`, city, address, state, id)
	if err != nil {
		return fmt.Errorf("patch pickup %s: %w", id, err)
	}
	return nil
}

func lotOrder(filter LotFilter, now time.Time) (string, []any) {
	dir := "ASC"
	if filter.Desc {
		dir = "DESC"
	}
	switch strings.ToLower(strings.TrimSpace(filter.Sort)) {
	case "kwara":
		return "ORDER BY kwara_id COLLATE NOCASE " + dir + ", CAST(number AS INTEGER) " + dir + ", id " + dir, nil
	case "title":
		return "ORDER BY title COLLATE NOCASE " + dir + ", id " + dir, nil
	case "city":
		return "ORDER BY pickup_city COLLATE NOCASE " + dir + ", pickup_address COLLATE NOCASE " + dir + ", id " + dir, nil
	case "bid":
		return "ORDER BY current_bid_cents " + dir + ", id " + dir, nil
	case "ends":
		return "ORDER BY CASE WHEN ends_at = '' THEN 1 ELSE 0 END ASC, datetime(ends_at) " + dir + ", id " + dir, nil
	default:
		return `ORDER BY
			CASE WHEN ended = 0 AND (ends_at = '' OR datetime(ends_at) > datetime(?)) THEN 0 ELSE 1 END,
			CASE WHEN ends_at = '' THEN 1 ELSE 0 END,
			datetime(ends_at) ASC,
			id ASC`, []any{now.UTC().Format(time.RFC3339)}
	}
}

func lotWhere(filter LotFilter, now time.Time) (string, []any) {
	clauses := []string{"1=1"}
	var args []any
	q := strings.TrimSpace(filter.Query)
	if q != "" {
		like := "%" + q + "%"
		clauses = append(clauses, "(title LIKE ? OR kwara_id LIKE ? OR number LIKE ? OR listing_title LIKE ? OR pickup_city LIKE ? OR pickup_address LIKE ?)")
		args = append(args, like, like, like, like, like, like)
	}
	if state := strings.TrimSpace(filter.PickupState); state != "" {
		clauses = append(clauses, "pickup_state = ?")
		args = append(args, state)
	}
	if filter.OpenOnly {
		clauses = append(clauses, "ended = 0 AND (ends_at = '' OR datetime(ends_at) > datetime(?))")
		args = append(args, now.UTC().Format(time.RFC3339))
	}
	return strings.Join(clauses, " AND "), args
}

func scanLots(rows *sql.Rows) ([]Lot, error) {
	var lots []Lot
	for rows.Next() {
		var lot Lot
		var ended int
		var synced string
		if err := rows.Scan(
			&lot.ID, &lot.Number, &lot.Title, &lot.Slug, &lot.ListingID, &lot.ListingTitle, &lot.KwaraID,
			&lot.CurrentBidCents, &lot.StartingBidCents, &lot.BidsCount, &lot.EndsAt,
			&lot.PickupState, &lot.PickupCity, &lot.PickupAddress, &ended, &lot.URL, &synced,
		); err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		lot.Ended = ended != 0
		if t, err := time.Parse("2006-01-02 15:04:05", synced); err == nil {
			lot.SyncedAt = t
		}
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan lots: %w", err)
	}
	if lots == nil {
		lots = []Lot{}
	}
	return lots, nil
}

func normalizeEndsAt(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return s
	}
	return t.UTC().Format(time.RFC3339)
}
