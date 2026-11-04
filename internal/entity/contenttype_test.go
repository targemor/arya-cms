package entity_test

import (
	"strings"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestContentTypeValidateValid(t *testing.T) {
	t.Parallel()

	attr := func(name string, typ entity.AttributeType) entity.Attribute {
		return entity.Attribute{Name: name, Type: typ}
	}
	title := attr("title", entity.AttributeTypeString)

	tests := []struct {
		name string
		ct   entity.ContentType
	}{
		{
			name: "valid collection type",
			ct: entity.ContentType{
				TableName: "articles",
				Attributes: []entity.Attribute{
					title,
					attr("views", entity.AttributeTypeInteger),
					attr("is_published", entity.AttributeTypeBoolean),
					attr("cover", entity.AttributeTypeMedia),
				},
			},
		},
		{
			name: "valid single type",
			ct: entity.ContentType{
				TableName:   "homepage",
				DisplayName: "Homepage Config",
				Kind:        entity.ContentTypeKindSingle,
				Attributes:  []entity.Attribute{title},
			},
		},
		{
			name: "valid component attribute",
			ct: entity.ContentType{
				TableName: "articles",
				Attributes: []entity.Attribute{
					{
						Name:       "contact",
						Type:       entity.AttributeTypeComponent,
						Component:  "shared.contact_data",
						Repeatable: true,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, tc.ct.Validate())
		})
	}
}

func TestContentTypeValidateErrors(t *testing.T) {
	t.Parallel()

	attr := func(name string, typ entity.AttributeType) entity.Attribute {
		return entity.Attribute{Name: name, Type: typ}
	}
	title := attr("title", entity.AttributeTypeString)
	withTitle := func(table string) entity.ContentType {
		return entity.ContentType{TableName: table, Attributes: []entity.Attribute{title}}
	}
	articles := func(attrs ...entity.Attribute) entity.ContentType {
		return entity.ContentType{TableName: "articles", Attributes: attrs}
	}

	tests := []struct {
		name string
		ct   entity.ContentType
	}{
		{name: "empty table name", ct: withTitle("")},
		{name: "sql injection in table name", ct: withTitle(`x"; DROP TABLE users; --`)},
		{name: "path traversal in table name", ct: withTitle("../etc")},
		{name: "uppercase table name", ct: withTitle("Articles")},
		{name: "table name starting with digit", ct: withTitle("1articles")},
		{name: "table name too long", ct: withTitle(strings.Repeat("a", 49))},
		{name: "no attributes", ct: articles()},
		{name: "invalid attribute name", ct: articles(attr("my title", entity.AttributeTypeString))},
		{name: "reserved attribute name", ct: articles(attr("created_at", entity.AttributeTypeString))},
		{name: "unsupported attribute type", ct: articles(attr("title", "date"))},
		{name: "duplicate attribute", ct: articles(title, attr("title", entity.AttributeTypeInteger))},
		{name: "component attribute missing component", ct: articles(entity.Attribute{
			Name: "contact",
			Type: entity.AttributeTypeComponent,
		})},
		{name: "invalid kind", ct: entity.ContentType{
			TableName:  "articles",
			Kind:       "invalid_kind",
			Attributes: []entity.Attribute{title},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.ct.Validate()
			require.ErrorIs(t, err, entity.ErrInvalidContentType)
		})
	}
}
