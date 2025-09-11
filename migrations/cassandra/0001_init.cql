-- Keyspace and tables for chat messages service (Cassandra/CQL)

CREATE KEYSPACE IF NOT EXISTS chat_keyspace
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
AND durable_writes = true;

USE chat_keyspace;

-- #############################################################################
-- # Core Tables
-- #############################################################################

-- Messages by room (partitioned by room_id, time-ordered by message_id)
CREATE TABLE IF NOT EXISTS messages_by_room (
    room_id uuid,
    message_id timeuuid,
    sender_id int,
    content text,
    content_decrypted text,
    type text,
    created_at timestamp,
    edited boolean,
    is_deleted boolean,
    reply_to_message_id timeuuid,
    forwarded_from_message_id timeuuid,
    file_url text,
    event text,
    sender_message_id text,
    PRIMARY KEY ((room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);

-- Rooms by user (denormalized list used for home view)
CREATE TABLE IF NOT EXISTS rooms_by_user (
    user_id int,
    is_pinned boolean,
    last_message_at timestamp,
    room_id uuid,
    room_name text,
    room_image text,
    room_type text,
    is_muted boolean,
    role text,
    last_message_id timeuuid,
    last_message_preview text,
    last_message_type text,
    last_message_sender_id int,
    last_message_sender_name text,
    last_message_sender_phone text,
    last_message_status int,
    last_message_updated_at timestamp,
    PRIMARY KEY ((user_id), is_pinned, last_message_at, room_id)
) WITH CLUSTERING ORDER BY (is_pinned DESC, last_message_at DESC, room_id DESC);

-- Unread counters by user and room
CREATE TABLE IF NOT EXISTS room_counters_by_user (
    user_id int,
    room_id uuid,
    unread_count counter,
    PRIMARY KEY ((user_id), room_id)
);

-- Room static details
CREATE TABLE IF NOT EXISTS room_details (
    room_id uuid PRIMARY KEY,
    name text,
    description text,
    image text,
    type text,
    encryption_data text,
    created_at timestamp,
    updated_at timestamp,
    join_all_user boolean,
    send_message boolean,
    add_member boolean,
    edit_group boolean
);

-- Room participants
CREATE TABLE IF NOT EXISTS participants_by_room (
    room_id uuid,
    user_id int,
    role text,
    joined_at timestamp,
    is_muted boolean,
    is_partner_blocked boolean,
    PRIMARY KEY ((room_id), user_id)
);

-- #############################################################################
-- # Metadata and Reverse Lookup Tables
-- #############################################################################

-- P2P rooms lookup to avoid duplicates
CREATE TABLE IF NOT EXISTS p2p_room_by_users (
    user1_id int,
    user2_id int,
    room_id uuid,
    PRIMARY KEY ((user1_id), user2_id)
);

-- Message reactions
CREATE TABLE IF NOT EXISTS reactions_by_message (
    message_id timeuuid,
    user_id int,
    reaction text,
    created_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);

-- Read receipts
CREATE TABLE IF NOT EXISTS read_receipts_by_message (
    message_id timeuuid,
    user_id int,
    read_at timestamp,
    PRIMARY KEY ((message_id), user_id)
);

-- Message lookup by client-provided id
CREATE TABLE IF NOT EXISTS message_by_sender_message_id (
    sender_message_id text PRIMARY KEY,
    room_id uuid,
    message_id timeuuid
);

-- Room lookup by message id
CREATE TABLE IF NOT EXISTS room_by_message (
    message_id timeuuid PRIMARY KEY,
    room_id uuid
);

-- Membership lookup
CREATE TABLE IF NOT EXISTS room_membership_lookup (
    user_id int,
    room_id uuid,
    is_pinned boolean,
    last_message_at timestamp,
    PRIMARY KEY ((user_id), room_id)
);

-- Deleted rooms per user (for sync)
CREATE TABLE IF NOT EXISTS deleted_rooms_by_user (
    user_id int,
    deleted_at timestamp,
    room_id uuid,
    reason text,
    PRIMARY KEY ((user_id), deleted_at, room_id)
) WITH CLUSTERING ORDER BY (deleted_at DESC);

-- Message status per user
CREATE TABLE IF NOT EXISTS message_status_by_user (
    user_id int,
    room_id uuid,
    message_id timeuuid,
    status int,
    PRIMARY KEY ((user_id, room_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
