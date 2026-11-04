ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) NOT NULL DEFAULT 'admin';
UPDATE users SET role = 'superadmin' WHERE email = 'admin@example.com';

