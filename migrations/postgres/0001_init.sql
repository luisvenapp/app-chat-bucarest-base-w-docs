-- Initial schema for chat messages service (PostgreSQL)
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Users table (minimal, for joins used by this service)
CREATE TABLE IF NOT EXISTS public."user" (
    id            INT PRIMARY KEY,
    name          TEXT NOT NULL,
    phone         TEXT UNIQUE,
    email         TEXT,
    avatar        TEXT,
    dni           TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    removed_at    TIMESTAMPTZ,
    deleted_at    TIMESTAMPTZ
);

-- Rooms table
CREATE TABLE IF NOT EXISTS public.room (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT,
    image            TEXT,
    description      TEXT,
    type             TEXT NOT NULL, -- p2p | group | channel
    encription_data  TEXT,          -- (sic) matches code spelling
    join_all_user    BOOLEAN DEFAULT FALSE,
    send_message     BOOLEAN DEFAULT TRUE,
    add_member       BOOLEAN DEFAULT FALSE,
    edit_group       BOOLEAN DEFAULT FALSE,
    "lastMessageAt"  TIMESTAMPTZ,   -- camelCase column used in queries
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_room_type ON public.room(type);
CREATE INDEX IF NOT EXISTS idx_room_last_message_at ON public.room("lastMessageAt");

-- Room members
CREATE TABLE IF NOT EXISTS public.room_member (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id             UUID NOT NULL REFERENCES public.room(id) ON DELETE CASCADE,
    user_id             INT  NOT NULL REFERENCES public."user"(id),
    role                TEXT NOT NULL DEFAULT 'MEMBER',
    "is_pinned"        BOOLEAN DEFAULT FALSE,
    "is_muted"         BOOLEAN DEFAULT FALSE,
    is_partner_blocked  BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW(),
    removed_at          TIMESTAMPTZ,
    deleted_at          TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_room_member_room_user ON public.room_member(room_id, user_id);
CREATE INDEX IF NOT EXISTS idx_room_member_user ON public.room_member(user_id);
CREATE INDEX IF NOT EXISTS idx_room_member_removed ON public.room_member(removed_at);

-- Messages
CREATE TABLE IF NOT EXISTS public.room_message (
    id                               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id                           UUID NOT NULL REFERENCES public.room(id) ON DELETE CASCADE,
    sender_id                         INT NOT NULL REFERENCES public."user"(id),
    content                           TEXT,
    content_decrypted                 TEXT,
    audio_transcription               TEXT,
    status                            INT DEFAULT 0,
    type                              TEXT DEFAULT 'message',
    lifetime                          TEXT,
    location_name                     TEXT,
    location_latitude                 DOUBLE PRECISION,
    location_longitude                DOUBLE PRECISION,
    origin                            TEXT,
    contact_id                        INT,
    contact_name                      TEXT,
    contact_phone                     TEXT,
    file                              TEXT,
    edited                            BOOLEAN DEFAULT FALSE,
    "isDeleted"                      BOOLEAN DEFAULT FALSE,
    replied_message_id                UUID,
    forwarded_message_id              UUID,
    forwarded_message_original_sender INT,
    event                             TEXT,
    sender_message_id                 TEXT,
    created_at                        TIMESTAMPTZ DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ DEFAULT NOW(),
    deleted_at                        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_room_message_room ON public.room_message(room_id);
CREATE INDEX IF NOT EXISTS idx_room_message_room_created ON public.room_message(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_room_message_sender ON public.room_message(sender_id);
CREATE INDEX IF NOT EXISTS idx_room_message_sender_msg_id ON public.room_message(sender_message_id);

-- Message metadata per user
CREATE TABLE IF NOT EXISTS public.room_message_meta (
    message_id         UUID NOT NULL REFERENCES public.room_message(id) ON DELETE CASCADE,
    user_id            INT  NOT NULL REFERENCES public."user"(id),
    read_at            TIMESTAMPTZ,
    "isDeleted"       BOOLEAN DEFAULT FALSE,
    "isSenderBlocked" BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_room_message_meta_user_read ON public.room_message_meta(user_id, read_at);

-- Message tags (mentions)
CREATE TABLE IF NOT EXISTS public.room_message_tag (
    message_id   UUID NOT NULL REFERENCES public.room_message(id) ON DELETE CASCADE,
    user_id      INT  NOT NULL REFERENCES public."user"(id),
    tag          TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    PRIMARY KEY (message_id, user_id, tag)
);

-- Message reactions (note camelCase columns used by code)
CREATE TABLE IF NOT EXISTS public.room_message_reaction (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "messageId" UUID NOT NULL REFERENCES public.room_message(id) ON DELETE CASCADE,
    "reactedById" INT NOT NULL REFERENCES public."user"(id),
    reaction     TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_reaction_message_user ON public.room_message_reaction("messageId", "reactedById") WHERE deleted_at IS NULL;

-- Tokens for push notifications
CREATE TABLE IF NOT EXISTS public.messaging_token (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token             TEXT NOT NULL,
    platform          TEXT,
    platform_version  TEXT,
    device            TEXT,
    lang              TEXT,
    is_voip           BOOLEAN DEFAULT FALSE,
    debug             BOOLEAN DEFAULT FALSE,
    user_id           INT NOT NULL REFERENCES public."user"(id),
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messaging_token_user ON public.messaging_token(user_id);

-- Helper views or functions could be added here if needed
