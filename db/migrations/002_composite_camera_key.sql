ALTER TABLE recordings
    DROP CONSTRAINT IF EXISTS recordings_camera_id_fkey;

ALTER TABLE recordings
    DROP CONSTRAINT IF EXISTS recordings_camera_fk;

ALTER TABLE cameras
    DROP CONSTRAINT IF EXISTS cameras_pkey;

ALTER TABLE cameras
    ADD PRIMARY KEY (truck_id, camera_id);

ALTER TABLE recordings
    ADD CONSTRAINT recordings_camera_fk
        FOREIGN KEY (truck_id, camera_id)
        REFERENCES cameras(truck_id, camera_id);
