package usecase_test

import (
	"context"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/component"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newComponentUseCase(t *testing.T) (usecase.Component, *MockComponentStore) {
	t.Helper()

	ctrl := gomock.NewController(t)
	store := NewMockComponentStore(ctrl)

	return component.New(store), store
}

func sampleComponent() entity.Component {
	return entity.Component{
		Name:        "contact_info",
		Category:    "shared",
		DisplayName: "Contact Info",
		Attributes: []entity.Attribute{
			{Name: "email", Type: entity.AttributeTypeString, Required: true},
		},
	}
}

func TestComponentCreate(t *testing.T) {
	t.Parallel()

	c := sampleComponent()

	tests := []struct {
		name    string
		input   entity.Component
		mock    func(s *MockComponentStore)
		wantErr error
	}{
		{
			name:  "success",
			input: c,
			mock: func(s *MockComponentStore) {
				s.EXPECT().Exists(gomock.Any(), "default", c.UID()).Return(false, nil)
				s.EXPECT().Save(gomock.Any(), "default", &c).Return(nil)
			},
		},
		{
			name:  "already exists",
			input: c,
			mock: func(s *MockComponentStore) {
				s.EXPECT().Exists(gomock.Any(), "default", c.UID()).Return(true, nil)
			},
			wantErr: entity.ErrComponentAlreadyExists,
		},
		{
			name: "invalid component",
			input: entity.Component{
				Name: "",
			},
			mock:    func(_ *MockComponentStore) {},
			wantErr: entity.ErrInvalidComponent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, store := newComponentUseCase(t)
			tc.mock(store)

			created, err := uc.Create(context.Background(), "default", &tc.input)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.input.Name, created.Name)
		})
	}
}

func TestComponentList(t *testing.T) {
	t.Parallel()

	c := sampleComponent()
	uc, store := newComponentUseCase(t)

	store.EXPECT().List(gomock.Any(), "default").Return([]entity.Component{c}, nil)

	comps, err := uc.List(context.Background(), "default")
	require.NoError(t, err)
	require.Len(t, comps, 1)
	require.Equal(t, c.Name, comps[0].Name)
}
