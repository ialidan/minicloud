-- Add media metadata columns for EXIF data extracted during upload.

ALTER TABLE files ADD COLUMN taken_at     TEXT;     -- EXIF DateTimeOriginal (RFC3339)
ALTER TABLE files ADD COLUMN camera_make  TEXT;     -- EXIF Make
ALTER TABLE files ADD COLUMN camera_model TEXT;     -- EXIF Model
ALTER TABLE files ADD COLUMN width        INTEGER;  -- pixels
ALTER TABLE files ADD COLUMN height       INTEGER;  -- pixels
ALTER TABLE files ADD COLUMN latitude     REAL;     -- GPS decimal degrees
ALTER TABLE files ADD COLUMN longitude    REAL;     -- GPS decimal degrees

-- Index for grouping/sorting media by capture date.
CREATE INDEX idx_files_taken_at ON files(owner_id, taken_at);
