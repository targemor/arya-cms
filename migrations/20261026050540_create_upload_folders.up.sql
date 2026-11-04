-- Media Library folders. A NULL parent_id is the library root.
CREATE TABLE IF NOT EXISTS upload_folders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    -- RESTRICT: a folder is only deleted once it is empty, so nothing is ever removed by cascade.
    parent_id  UUID REFERENCES upload_folders (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One name per parent; NULLS NOT DISTINCT makes it hold at the root too.
    CONSTRAINT upload_folders_parent_name_key UNIQUE NULLS NOT DISTINCT (parent_id, name),
    CONSTRAINT upload_folders_not_own_parent CHECK (parent_id <> id)
);

CREATE INDEX IF NOT EXISTS upload_folders_parent_idx ON upload_folders (parent_id);

-- A NULL folder_id is the library root; existing files stay there.
ALTER TABLE upload_files ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES upload_folders (id) ON DELETE RESTRICT;

-- Listing a folder, newest first.
CREATE INDEX IF NOT EXISTS upload_files_folder_idx ON upload_files (folder_id, created_at DESC, id);
