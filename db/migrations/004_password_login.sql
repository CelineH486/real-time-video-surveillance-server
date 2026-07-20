CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT;

CREATE INDEX IF NOT EXISTS user_api_tokens_token_user_idx
    ON user_api_tokens (token_hash, user_id);
