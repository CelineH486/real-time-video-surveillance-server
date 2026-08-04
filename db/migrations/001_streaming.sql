ALTER TABLE cameras
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS recordings (
    recording_id BIGSERIAL PRIMARY KEY,
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id),
    camera_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ready',
    CONSTRAINT recordings_camera_fk
        FOREIGN KEY (truck_id, camera_id)
        REFERENCES cameras(truck_id, camera_id)
);

CREATE INDEX IF NOT EXISTS recordings_lookup_idx
    ON recordings (truck_id, camera_id, started_at DESC);
