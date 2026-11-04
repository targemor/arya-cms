CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

INSERT INTO projects (id, name, slug, description)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project', 'default', 'Default project')
ON CONFLICT (slug) DO NOTHING;

