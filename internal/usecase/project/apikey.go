package project

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
)

const (
	_apiKeyBytes     = 24
	_apiKeyPrefixLen = 12
)

func generateRawAPIKey() (raw, prefix, hash string, err error) {
	buf := make([]byte, _apiKeyBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", err
	}

	raw = "cms_" + hex.EncodeToString(buf)
	prefix = raw[:_apiKeyPrefixLen]
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])

	return raw, prefix, hash, nil
}

// CreateAPIKey generates a new project-scoped API key and persists its SHA-256 hash.
func (uc *UseCase) CreateAPIKey(
	ctx context.Context,
	slug, name, description string,
	accessType entity.APIKeyAccessType,
	permissions map[string][]entity.APIKeyAction,
) (entity.APIKey, error) {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - CreateAPIKey - uc.repo.GetBySlug: %w", err)
	}

	raw, prefix, hash, err := generateRawAPIKey()
	if err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - CreateAPIKey - generateRawAPIKey: %w", err)
	}

	k := entity.APIKey{
		ID:          uuid.New(),
		ProjectID:   p.ID,
		ProjectSlug: p.Slug,
		Name:        name,
		Description: description,
		KeyPrefix:   prefix,
		KeyHash:     hash,
		Token:       raw,
		AccessType:  accessType,
		Permissions: permissions,
	}

	if err = k.Validate(); err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - CreateAPIKey - k.Validate: %w", err)
	}

	if err = uc.repo.CreateAPIKey(ctx, &k); err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - CreateAPIKey - uc.repo.CreateAPIKey: %w", err)
	}

	return k, nil
}

// ListAPIKeys returns all API keys configured for a project.
func (uc *UseCase) ListAPIKeys(ctx context.Context, slug string) ([]entity.APIKey, error) {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - ListAPIKeys - uc.repo.GetBySlug: %w", err)
	}

	keys, err := uc.repo.ListAPIKeys(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - ListAPIKeys - uc.repo.ListAPIKeys: %w", err)
	}

	if keys == nil {
		keys = []entity.APIKey{}
	}

	return keys, nil
}

// UpdateAPIKey updates an API key's name, description, access type, and granular permissions.
func (uc *UseCase) UpdateAPIKey(
	ctx context.Context,
	slug, keyID, name, description string,
	accessType entity.APIKeyAccessType,
	permissions map[string][]entity.APIKeyAction,
) (entity.APIKey, error) {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - UpdateAPIKey - uc.repo.GetBySlug: %w", err)
	}

	parsedID, err := uuid.Parse(keyID)
	if err != nil {
		return entity.APIKey{}, fmt.Errorf("%w: invalid api key id", entity.ErrInvalidAPIKey)
	}

	k := entity.APIKey{
		ID:          parsedID,
		ProjectID:   p.ID,
		ProjectSlug: p.Slug,
		Name:        name,
		Description: description,
		AccessType:  accessType,
		Permissions: permissions,
	}

	if err = k.Validate(); err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - UpdateAPIKey - k.Validate: %w", err)
	}

	if err = uc.repo.UpdateAPIKey(ctx, &k); err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - UpdateAPIKey - uc.repo.UpdateAPIKey: %w", err)
	}

	return k, nil
}

// DeleteAPIKey revokes/deletes an API key from a project.
func (uc *UseCase) DeleteAPIKey(ctx context.Context, slug, keyID string) error {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("ProjectUseCase - DeleteAPIKey - uc.repo.GetBySlug: %w", err)
	}

	parsedID, err := uuid.Parse(keyID)
	if err != nil {
		return fmt.Errorf("%w: invalid api key id", entity.ErrInvalidAPIKey)
	}

	if err = uc.repo.DeleteAPIKey(ctx, p.ID, parsedID); err != nil {
		return fmt.Errorf("ProjectUseCase - DeleteAPIKey - uc.repo.DeleteAPIKey: %w", err)
	}

	return nil
}

// ValidateAPIKey hashes rawKey, looks up the active API key, and updates its last_used_at timestamp.
func (uc *UseCase) ValidateAPIKey(ctx context.Context, rawKey string) (entity.APIKey, error) {
	if rawKey == "" {
		return entity.APIKey{}, fmt.Errorf("%w: empty api key", entity.ErrInvalidAPIKey)
	}

	sum := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(sum[:])

	k, err := uc.repo.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return entity.APIKey{}, fmt.Errorf("ProjectUseCase - ValidateAPIKey - uc.repo.GetAPIKeyByHash: %w", err)
	}

	if touchErr := uc.repo.TouchAPIKey(ctx, k.ID); touchErr != nil {
		return *k, nil
	}

	return *k, nil
}
