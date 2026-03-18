CREATE INDEX IF NOT EXISTS idx_files_owner_checksum ON files(owner_id, checksum);
