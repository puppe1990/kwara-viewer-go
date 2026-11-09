-- 001_init.sql — compact Kwara lots cached locally.
-- up
CREATE TABLE IF NOT EXISTS lots (
  id TEXT PRIMARY KEY,
  number TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  listing_id TEXT NOT NULL DEFAULT '',
  listing_title TEXT NOT NULL DEFAULT '',
  kwara_id TEXT NOT NULL DEFAULT '',
  current_bid_cents INTEGER NOT NULL DEFAULT 0,
  starting_bid_cents INTEGER NOT NULL DEFAULT 0,
  bids_count INTEGER NOT NULL DEFAULT 0,
  ends_at TEXT NOT NULL DEFAULT '',
  pickup_state TEXT NOT NULL DEFAULT '',
  ended INTEGER NOT NULL DEFAULT 0,
  url TEXT NOT NULL DEFAULT '',
  synced_at TEXT NOT NULL DEFAULT (datetime('now')),
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS lots_pickup_state ON lots (pickup_state);
CREATE INDEX IF NOT EXISTS lots_ends_at ON lots (ends_at);
CREATE INDEX IF NOT EXISTS lots_ended ON lots (ended);
CREATE INDEX IF NOT EXISTS lots_kwara_id ON lots (kwara_id);

-- down
DROP INDEX IF EXISTS lots_kwara_id;
DROP INDEX IF EXISTS lots_ended;
DROP INDEX IF EXISTS lots_ends_at;
DROP INDEX IF EXISTS lots_pickup_state;
DROP TABLE IF EXISTS lots;
