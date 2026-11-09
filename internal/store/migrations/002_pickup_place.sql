-- 002_pickup_place.sql — city and venue from Kwara listing.location / pickUpAddress.
-- up
ALTER TABLE lots ADD COLUMN pickup_city TEXT NOT NULL DEFAULT '';
ALTER TABLE lots ADD COLUMN pickup_address TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS lots_pickup_city ON lots (pickup_city);

-- down
DROP INDEX IF EXISTS lots_pickup_city;
ALTER TABLE lots DROP COLUMN pickup_address;
ALTER TABLE lots DROP COLUMN pickup_city;
