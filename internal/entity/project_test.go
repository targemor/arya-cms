package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectValidate(t *testing.T) {
	t.Parallel()

	valid := entity.Project{
		ID:   uuid.New(),
		Name: "Main Store",
		Slug: "main_store",
	}
	require.NoError(t, valid.Validate())

	emptyName := valid
	emptyName.Name = ""
	assert.ErrorIs(t, emptyName.Validate(), entity.ErrInvalidProject)

	badSlug := valid
	badSlug.Slug = "Main-Store!"
	assert.ErrorIs(t, badSlug.Validate(), entity.ErrInvalidProject)
}
