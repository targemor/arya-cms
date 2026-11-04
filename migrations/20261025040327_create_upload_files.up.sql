-- Media Library files, Strapi style. The bytes live wherever "provider" put them; this row is the metadata.
CREATE TABLE IF NOT EXISTS upload_files (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,             -- original file name, as uploaded
    alternative_text TEXT,                              -- alt text for images
    hash             VARCHAR(255) NOT NULL UNIQUE,      -- generated unique name; the stored object is hash || ext
    ext              VARCHAR(32)  NOT NULL DEFAULT '',  -- ".png", including the dot; '' when the file had none
    mime             VARCHAR(255) NOT NULL,
    size             BIGINT       NOT NULL CHECK (size >= 0), -- bytes
    url              TEXT         NOT NULL,             -- public URL returned by the provider
    provider         VARCHAR(64)  NOT NULL,             -- "local", "aws-s3", ...
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- The library lists newest first.
CREATE INDEX IF NOT EXISTS upload_files_created_at_idx ON upload_files (created_at DESC, id);
