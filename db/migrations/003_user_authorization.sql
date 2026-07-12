CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_api_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS user_api_tokens_user_id_idx
    ON user_api_tokens (user_id);

CREATE TABLE IF NOT EXISTS user_trucks (
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    truck_id TEXT NOT NULL REFERENCES trucks(truck_id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, truck_id)
);

CREATE INDEX IF NOT EXISTS user_trucks_truck_id_idx
    ON user_trucks (truck_id);
