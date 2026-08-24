CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname      TEXT NOT NULL DEFAULT '',
    avatar_url    TEXT NOT NULL DEFAULT '',
    home_water    TEXT NOT NULL DEFAULT '',
    lure_pref     TEXT NOT NULL DEFAULT '',
    credit_score  INTEGER NOT NULL DEFAULT 100,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS clubs (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    owner_id    UUID NOT NULL REFERENCES users(id),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS club_members (
    club_id    UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('OWNER','ADMIN','MEMBER')),
    status     TEXT NOT NULL CHECK (status IN ('PENDING','APPROVED','REJECTED')),
    joined_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (club_id, user_id)
);

CREATE TABLE IF NOT EXISTS friendships (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (status IN ('PENDING','ACCEPTED')),
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, friend_id)
);

CREATE TABLE IF NOT EXISTS spots (
    id            UUID PRIMARY KEY,
    owner_id      UUID NOT NULL REFERENCES users(id),
    club_id       UUID REFERENCES clubs(id),
    name          TEXT NOT NULL,
    water_type    TEXT NOT NULL,
    structure     TEXT NOT NULL,
    visibility    TEXT NOT NULL CHECK (visibility IN ('PRIVATE','CLUB','FRIENDS','PUBLIC')),
    location      geography(Point, 4326) NOT NULL,
    lat           DOUBLE PRECISION NOT NULL,
    lon           DOUBLE PRECISION NOT NULL,
    shore_bearing DOUBLE PRECISION NOT NULL DEFAULT 90,
    tidal         BOOLEAN NOT NULL DEFAULT FALSE,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_spots_geo ON spots USING GIST (location);
CREATE INDEX IF NOT EXISTS idx_spots_owner ON spots (owner_id);
CREATE INDEX IF NOT EXISTS idx_spots_visibility ON spots (visibility);

CREATE TABLE IF NOT EXISTS spot_depths (
    id         UUID PRIMARY KEY,
    spot_id    UUID NOT NULL REFERENCES spots(id) ON DELETE CASCADE,
    offset_m   DOUBLE PRECISION NOT NULL DEFAULT 0,
    depth_m    DOUBLE PRECISION NOT NULL,
    sampled_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS catches (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    spot_id         UUID NOT NULL REFERENCES spots(id),
    caught_at       TIMESTAMPTZ NOT NULL,
    timezone        TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    local_time      TEXT NOT NULL,
    species         TEXT NOT NULL,
    length_cm       DOUBLE PRECISION NOT NULL,
    weight_kg       DOUBLE PRECISION,
    lure_type       TEXT NOT NULL,
    lure_weight_g   DOUBLE PRECISION,
    lure_color      TEXT NOT NULL DEFAULT '',
    retrieve        TEXT NOT NULL DEFAULT '',
    layer           TEXT NOT NULL DEFAULT '',
    water_depth_m   DOUBLE PRECISION,
    water_color     TEXT NOT NULL DEFAULT '',
    turbidity       TEXT NOT NULL DEFAULT '',
    water_temp_c    DOUBLE PRECISION,
    current         TEXT NOT NULL DEFAULT '',
    released        BOOLEAN NOT NULL DEFAULT TRUE,
    note            TEXT NOT NULL DEFAULT '',
    photo_key       TEXT NOT NULL DEFAULT '',
    hydro_status    TEXT NOT NULL DEFAULT 'PENDING',
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catches_user_time ON catches (user_id, caught_at DESC);
CREATE INDEX IF NOT EXISTS idx_catches_spot ON catches (spot_id);

CREATE TABLE IF NOT EXISTS hydro_jobs (
    id            UUID PRIMARY KEY,
    catch_id      UUID NOT NULL UNIQUE REFERENCES catches(id) ON DELETE CASCADE,
    status        TEXT NOT NULL CHECK (status IN ('PENDING','RUNNING','BOUND','FAILED')),
    attempts      INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT NOT NULL DEFAULT '',
    error_class   TEXT NOT NULL DEFAULT '',
    next_run_at   TIMESTAMPTZ NOT NULL,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_hydro_jobs_due ON hydro_jobs (status, next_run_at);

CREATE TABLE IF NOT EXISTS hydro_snapshots (
    catch_id          UUID PRIMARY KEY REFERENCES catches(id) ON DELETE CASCADE,
    bound_at          TIMESTAMPTZ NOT NULL,
    bind_error_sec    INTEGER NOT NULL DEFAULT 0,
    pressure_hpa      DOUBLE PRECISION NOT NULL,
    pressure_delta_3h DOUBLE PRECISION NOT NULL,
    pressure_trend    TEXT NOT NULL,
    air_temp_c        DOUBLE PRECISION NOT NULL,
    wind_dir_deg      DOUBLE PRECISION NOT NULL,
    wind_dir_label    TEXT NOT NULL,
    wind_speed_ms     DOUBLE PRECISION NOT NULL,
    beaufort          INTEGER NOT NULL,
    shore_aspect      TEXT NOT NULL,
    tide_height_m     DOUBLE PRECISION NOT NULL,
    tide_phase_pct    DOUBLE PRECISION NOT NULL,
    tide_window       TEXT NOT NULL,
    moon_phase        TEXT NOT NULL,
    moon_illum_pct    DOUBLE PRECISION NOT NULL,
    bite_score        DOUBLE PRECISION NOT NULL,
    frenzy            BOOLEAN NOT NULL DEFAULT FALSE,
    contributions     JSONB NOT NULL DEFAULT '[]',
    series            JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS activities (
    id           UUID PRIMARY KEY,
    host_id      UUID NOT NULL REFERENCES users(id),
    club_id      UUID REFERENCES clubs(id),
    spot_id      UUID NOT NULL REFERENCES spots(id),
    title        TEXT NOT NULL,
    kind         TEXT NOT NULL CHECK (kind IN ('CLUB_CHARTER','WILD_MEETUP','HOT_SLOT')),
    status       TEXT NOT NULL CHECK (status IN ('DRAFT','OPEN','CLOSED','CANCELLED')),
    starts_at    TIMESTAMPTZ NOT NULL,
    ends_at      TIMESTAMPTZ NOT NULL,
    meet_lat     DOUBLE PRECISION NOT NULL,
    meet_lon     DOUBLE PRECISION NOT NULL,
    meet_radius_m DOUBLE PRECISION NOT NULL DEFAULT 300,
    fee_amount   NUMERIC(10,2) NOT NULL DEFAULT 0,
    fee_note     TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS slots (
    id              UUID PRIMARY KEY,
    activity_id     UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    label           TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('OPEN','LOCKED','CONFIRMED','CHECKED_IN')),
    holder_id       UUID REFERENCES users(id),
    locked_at       TIMESTAMPTZ,
    lock_expires_at TIMESTAMPTZ,
    confirmed_at    TIMESTAMPTZ,
    checked_in_at   TIMESTAMPTZ,
    version         INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_slot_active_holder
    ON slots (id)
    WHERE status IN ('LOCKED','CONFIRMED','CHECKED_IN') AND holder_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS slot_holds (
    slot_id    UUID PRIMARY KEY REFERENCES slots(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id),
    status     TEXT NOT NULL CHECK (status IN ('LOCKED','CONFIRMED','CHECKED_IN')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS checkins (
    id          UUID PRIMARY KEY,
    activity_id UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    distance_m  DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (activity_id, user_id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id         UUID PRIMARY KEY,
    user_id    UUID,
    action     TEXT NOT NULL,
    target     TEXT NOT NULL,
    detail     JSONB NOT NULL DEFAULT '{}',
    ip         TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_user_action ON audit_logs (user_id, action, created_at DESC);

CREATE TABLE IF NOT EXISTS user_stats (
    user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_catches  INTEGER NOT NULL DEFAULT 0,
    released_count INTEGER NOT NULL DEFAULT 0,
    max_length_cm  DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_species    TEXT NOT NULL DEFAULT '',
    streak_days    INTEGER NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL
);
