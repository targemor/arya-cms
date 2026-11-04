CREATE TABLE IF NOT EXISTS project_api_keys (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    access_type VARCHAR(32) NOT NULL DEFAULT 'custom',
    permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_project_api_keys_project_id ON project_api_keys(project_id);
CREATE INDEX IF NOT EXISTS idx_project_api_keys_key_hash ON project_api_keys(key_hash);

