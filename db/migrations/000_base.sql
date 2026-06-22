CREATE TABLE IF NOT EXISTS trucks (
    truck_id TEXT PRIMARY KEY,
    plate_no TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline'
);

CREATE TABLE IF NOT EXISTS cameras (
    camera_id TEXT PRIMARY KEY,
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline'
);

CREATE INDEX IF NOT EXISTS cameras_truck_id_idx ON cameras (truck_id);
