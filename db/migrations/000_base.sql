CREATE TABLE IF NOT EXISTS trucks (
    truck_id TEXT PRIMARY KEY,
    plate_no TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline'
);

CREATE TABLE IF NOT EXISTS cameras (
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id) ON DELETE CASCADE,
    camera_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline',
    PRIMARY KEY (truck_id, camera_id)
);

CREATE INDEX IF NOT EXISTS cameras_truck_id_idx ON cameras (truck_id);
