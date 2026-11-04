package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lucimux/arya/internal/entity"
)

// CreateAPIKey inserts a new project API key.
func (r *Repo) CreateAPIKey(ctx context.Context, k *entity.APIKey) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}

	permsJSON, err := json.Marshal(k.Permissions)
	if err != nil {
		return fmt.Errorf("ProjectRepo - CreateAPIKey - json.Marshal: %w", err)
	}

	query := `
		INSERT INTO project_api_keys (id, project_id, name, description, key_prefix, key_hash, access_type, permissions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, project_id, '' AS slug, name, description, key_prefix, key_hash, access_type, permissions, last_used_at, created_at, updated_at`

	token := k.Token
	slug := k.ProjectSlug

	created, err := scanAPIKeyRow(r.Pool.QueryRow(
		ctx,
		query,
		k.ID,
		k.ProjectID,
		k.Name,
		k.Description,
		k.KeyPrefix,
		k.KeyHash,
		string(k.AccessType),
		permsJSON,
	))
	if err != nil {
		return fmt.Errorf("ProjectRepo - CreateAPIKey - scanAPIKeyRow: %w", err)
	}

	created.Token = token
	created.ProjectSlug = slug
	*k = created

	return nil
}

// ListAPIKeys returns all API keys belonging to a project.
func (r *Repo) ListAPIKeys(ctx context.Context, projectID uuid.UUID) ([]entity.APIKey, error) {
	query := `
		SELECT k.id, k.project_id, p.slug, k.name, k.description, k.key_prefix, k.key_hash, k.access_type, k.permissions, k.last_used_at, k.created_at, k.updated_at
		FROM project_api_keys k
		INNER JOIN projects p ON k.project_id = p.id
		WHERE k.project_id = $1
		ORDER BY k.created_at DESC`

	rows, err := r.Pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - ListAPIKeys - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	var keys []entity.APIKey

	for rows.Next() {
		k, scanErr := scanAPIKeyRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ProjectRepo - ListAPIKeys - scanAPIKeyRow: %w", scanErr)
		}

		keys = append(keys, k)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("ProjectRepo - ListAPIKeys - rows.Err: %w", rows.Err())
	}

	return keys, nil
}

// GetAPIKeyByHash retrieves an API key and its project slug by SHA-256 hash.
func (r *Repo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	query := `
		SELECT k.id, k.project_id, p.slug, k.name, k.description, k.key_prefix, k.key_hash, k.access_type, k.permissions, k.last_used_at, k.created_at, k.updated_at
		FROM project_api_keys k
		INNER JOIN projects p ON k.project_id = p.id
		WHERE k.key_hash = $1`

	k, err := scanAPIKeyRow(r.Pool.QueryRow(ctx, query, keyHash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAPIKeyNotFound
		}

		return nil, fmt.Errorf("ProjectRepo - GetAPIKeyByHash - scanAPIKeyRow: %w", err)
	}

	return &k, nil
}

// UpdateAPIKey updates an API key's metadata, access type, and permissions.
func (r *Repo) UpdateAPIKey(ctx context.Context, k *entity.APIKey) error {
	permsJSON, err := json.Marshal(k.Permissions)
	if err != nil {
		return fmt.Errorf("ProjectRepo - UpdateAPIKey - json.Marshal: %w", err)
	}

	query := `
		UPDATE project_api_keys
		SET name = $1, description = $2, access_type = $3, permissions = $4, updated_at = now()
		WHERE id = $5 AND project_id = $6
		RETURNING id, project_id, '' AS slug, name, description, key_prefix, key_hash, access_type, permissions, last_used_at, created_at, updated_at`

	slug := k.ProjectSlug

	updated, err := scanAPIKeyRow(r.Pool.QueryRow(
		ctx,
		query,
		k.Name,
		k.Description,
		string(k.AccessType),
		permsJSON,
		k.ID,
		k.ProjectID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.ErrAPIKeyNotFound
		}

		return fmt.Errorf("ProjectRepo - UpdateAPIKey - scanAPIKeyRow: %w", err)
	}

	updated.ProjectSlug = slug
	*k = updated

	return nil
}

// DeleteAPIKey deletes an API key belonging to a project.
func (r *Repo) DeleteAPIKey(ctx context.Context, projectID, keyID uuid.UUID) error {
	query := `DELETE FROM project_api_keys WHERE id = $1 AND project_id = $2`

	tag, err := r.Pool.Exec(ctx, query, keyID, projectID)
	if err != nil {
		return fmt.Errorf("ProjectRepo - DeleteAPIKey - r.Pool.Exec: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrAPIKeyNotFound
	}

	return nil
}

// TouchAPIKey updates last_used_at timestamp for an API key.
func (r *Repo) TouchAPIKey(ctx context.Context, keyID uuid.UUID) error {
	query := `UPDATE project_api_keys SET last_used_at = now() WHERE id = $1`

	if _, err := r.Pool.Exec(ctx, query, keyID); err != nil {
		return fmt.Errorf("ProjectRepo - TouchAPIKey - r.Pool.Exec: %w", err)
	}

	return nil
}

func scanAPIKeyRow(s rowScanner) (entity.APIKey, error) {
	var (
		k           entity.APIKey
		description *string
		accessType  string
		permsRaw    []byte
		lastUsedAt  *time.Time
	)

	err := s.Scan(
		&k.ID,
		&k.ProjectID,
		&k.ProjectSlug,
		&k.Name,
		&description,
		&k.KeyPrefix,
		&k.KeyHash,
		&accessType,
		&permsRaw,
		&lastUsedAt,
		&k.CreatedAt,
		&k.UpdatedAt,
	)
	if err != nil {
		return entity.APIKey{}, err
	}

	if description != nil {
		k.Description = *description
	}

	k.AccessType = entity.APIKeyAccessType(accessType)
	k.LastUsedAt = lastUsedAt
	k.Permissions = make(map[string][]entity.APIKeyAction)

	if len(permsRaw) > 0 {
		if err := json.Unmarshal(permsRaw, &k.Permissions); err != nil {
			return entity.APIKey{}, err
		}
	}

	return k, nil
}
