CREATE TABLE IF NOT EXISTS project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL DEFAULT 'editor',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);

INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, u.id, u.role
FROM projects p
CROSS JOIN users u
WHERE p.slug = 'default'
ON CONFLICT DO NOTHING;

