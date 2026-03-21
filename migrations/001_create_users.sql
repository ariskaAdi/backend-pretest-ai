CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('admin', 'member', 'guest');

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100)        NOT NULL,
    email       VARCHAR(150) UNIQUE NOT NULL,
    password    VARCHAR(255)        NOT NULL,
    role        user_role           NOT NULL DEFAULT 'guest',
    otp         VARCHAR(6),
    is_verified BOOLEAN             NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
