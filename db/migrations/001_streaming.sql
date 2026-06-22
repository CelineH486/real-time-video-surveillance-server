ALTER TABLE cameras
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS recordings (
    recording_id BIGSERIAL PRIMARY KEY,
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id),
    camera_id TEXT NOT NULL REFERENCES cameras(camera_id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ready'
);

CREATE INDEX IF NOT EXISTS recordings_lookup_idx
    ON recordings (truck_id, camera_id, started_at DESC);
