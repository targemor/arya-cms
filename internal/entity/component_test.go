package entity_test

import (
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestComponentValidate(t *testing.T) {
	t.Parallel()

	title := entity.Attribute{Name: "title", Type: entity.AttributeTypeString}

	tests := []struct {
		name    string
		comp    entity.Component
		wantErr bool
	}{
		{
			name: "valid component",
			comp: entity.Component{
				Name:        "contact_info",
				Category:    "shared",
				DisplayName: "Contact Info",
				Attributes:  []entity.Attribute{title},
			},
		},
		{
			name: "invalid name",
			comp: entity.Component{
				Name:       "Invalid-Name",
				Category:   "shared",
				Attributes: []entity.Attribute{title},
			},
			wantErr: true,
		},
		{
			name: "invalid category",
			comp: entity.Component{
				Name:       "contact_info",
				Category:   "Invalid Category",
				Attributes: []entity.Attribute{title},
			},
			wantErr: true,
		},
		{
			name: "empty attributes",
			comp: entity.Component{
				Name:       "contact_info",
				Category:   "shared",
				Attributes: []entity.Attribute{},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.comp.Validate()
			if tc.wantErr {
				require.ErrorIs(t, err, entity.ErrInvalidComponent)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.comp.Category+"."+tc.comp.Name, tc.comp.UID())
		})
	}
}
