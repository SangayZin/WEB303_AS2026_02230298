-- checkins_db schema
CREATE TABLE IF NOT EXISTS checkins (
    id            UUID PRIMARY KEY,
    member_id     UUID NOT NULL,
    check_in_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checkins_member_id  ON checkins(member_id);
CREATE INDEX IF NOT EXISTS idx_checkins_check_in_at ON checkins(check_in_at DESC);
