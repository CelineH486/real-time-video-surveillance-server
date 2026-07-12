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
ON CONFLICT (truck_id, camera_id) DO NOTHING;

INSERT INTO users (user_id, email, display_name, status)
VALUES ('dev-user', 'dev@example.local', 'Development User', 'active')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_trucks (user_id, truck_id, role)
VALUES ('dev-user', 'truck001', 'viewer')
ON CONFLICT (user_id, truck_id) DO NOTHING;

-- Plaintext development token: dev-user-token
INSERT INTO user_api_tokens (token_hash, user_id, label)
VALUES ('ff194a51405eb34180b91ed9d9130ec5ddec108174c6806fc333ec3c33d83870', 'dev-user', 'Local development')
ON CONFLICT (token_hash) DO NOTHING;
