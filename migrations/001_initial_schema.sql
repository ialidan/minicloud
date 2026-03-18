-- Initial schema: users, files, sessions.

CREATE TABLE users (
    id           TEXT PRIMARY KEY,
    username     TEXT NOT NULL UNIQUE COLLATE NOCASE,
    email        TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin', 'user')),
    is_active    INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE files (
    id            TEXT PRIMARY KEY,
    owner_id      TEXT NOT NULL,
    virtual_path  TEXT NOT NULL DEFAULT '/',
    original_name TEXT NOT NULL,
    storage_name  TEXT NOT NULL UNIQUE,
    size          INTEGER NOT NULL,
    mime_type     TEXT NOT NULL DEFAULT 'application/octet-stream',
    checksum      TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_files_owner_path ON files(owner_id, virtual_path);
CREATE UNIQUE INDEX idx_files_owner_name_path ON files(owner_id, virtual_path, original_name);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
