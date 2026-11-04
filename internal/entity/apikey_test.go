package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyValidateAndCanPerform(t *testing.T) {
	t.Parallel()

	key := entity.APIKey{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		Name:       "Mobile App",
		AccessType: entity.APIKeyAccessCustom,
		Permissions: map[string][]entity.APIKeyAction{
			"articles": {entity.APIKeyActionFind, entity.APIKeyActionFindOne},
			"orders":   {entity.APIKeyActionCreate},
		},
	}

	require.NoError(t, key.Validate())
	assert.True(t, key.CanPerform("articles", entity.APIKeyActionFind))
	assert.True(t, key.CanPerform("articles", entity.APIKeyActionFindOne))
	assert.False(t, key.CanPerform("articles", entity.APIKeyActionDelete))
	assert.True(t, key.CanPerform("orders", entity.APIKeyActionCreate))
	assert.False(t, key.CanPerform("orders", entity.APIKeyActionFind))
	assert.False(t, key.CanPerform("unknown", entity.APIKeyActionFind))

	readOnlyKey := entity.APIKey{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		Name:       "Public Web",
		AccessType: entity.APIKeyAccessReadOnly,
	}
	require.NoError(t, readOnlyKey.Validate())
	assert.True(t, readOnlyKey.CanPerform("any_table", entity.APIKeyActionFind))
	assert.True(t, readOnlyKey.CanPerform("any_table", entity.APIKeyActionFindOne))
	assert.False(t, readOnlyKey.CanPerform("any_table", entity.APIKeyActionCreate))

	fullKey := entity.APIKey{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		Name:       "Admin Service",
		AccessType: entity.APIKeyAccessFull,
	}
	require.NoError(t, fullKey.Validate())
	assert.True(t, fullKey.CanPerform("any_table", entity.APIKeyActionDelete))
}
