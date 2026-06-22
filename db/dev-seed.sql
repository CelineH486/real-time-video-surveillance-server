INSERT INTO trucks (truck_id, plate_no, status)
VALUES ('truck001', 'TEST-001', 'offline')
ON CONFLICT (truck_id) DO NOTHING;

INSERT INTO cameras (camera_id, truck_id, name, status)
SELECT
    'cam' || LPAD(camera_number::TEXT, 2, '0'),
    'truck001',
    'Camera ' || camera_number,
    'offline'
FROM generate_series(1, 9) AS camera_number
ON CONFLICT (camera_id) DO NOTHING;
