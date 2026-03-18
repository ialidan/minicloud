-- Explicit directory entries for the virtual filesystem.

CREATE TABLE directories (
    id          TEXT PRIMARY KEY,
    owner_id    TEXT NOT NULL,
    parent_path TEXT NOT NULL DEFAULT '/',
    name        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_directories_owner_parent ON directories(owner_id, parent_path);
CREATE UNIQUE INDEX idx_directories_owner_parent_name ON directories(owner_id, parent_path, name);
