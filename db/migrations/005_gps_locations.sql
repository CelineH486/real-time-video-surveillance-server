CREATE TABLE IF NOT EXISTS truck_locations (
    location_id BIGSERIAL PRIMARY KEY,
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    altitude_m DOUBLE PRECISION,
    speed_kmh DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (speed_kmh >= 0),
    heading_degrees DOUBLE PRECISION CHECK (heading_degrees >= 0 AND heading_degrees < 360),
    accuracy_m DOUBLE PRECISION CHECK (accuracy_m >= 0),
    satellites INTEGER CHECK (satellites >= 0),
    fix_quality INTEGER NOT NULL CHECK (fix_quality > 0),
    recorded_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS truck_locations_truck_time_idx
    ON truck_locations (truck_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS truck_location_state (
    truck_id TEXT PRIMARY KEY REFERENCES trucks(truck_id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    altitude_m DOUBLE PRECISION,
    speed_kmh DOUBLE PRECISION NOT NULL,
    heading_degrees DOUBLE PRECISION,
    accuracy_m DOUBLE PRECISION,
    satellites INTEGER,
    fix_quality INTEGER NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    stop_anchor_latitude DOUBLE PRECISION,
    stop_anchor_longitude DOUBLE PRECISION,
    stopped_since TIMESTAMPTZ
);
