-- members_db schema
CREATE TABLE IF NOT EXISTS members (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    phone      TEXT NOT NULL,
    plan       TEXT NOT NULL CHECK (plan IN ('basic','premium','vip')),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_members_email ON members(email);
