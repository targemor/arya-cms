package content

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toSQL(t *testing.T, s sq.Sqlizer) (sql string, args []any) {
	t.Helper()

	sql, args, err := s.ToSql()
	require.NoError(t, err)

	return sql, args
}

func TestCondition(t *testing.T) {
	t.Parallel()

	filter := func(field string, op entity.FilterOperator, values ...any) entity.Filter {
		return entity.Filter{Field: field, Operator: op, Values: values}
	}

	tests := []struct {
		name     string
		filter   entity.Filter
		wantSQL  string
		wantArgs []any
	}{
		{"eq", filter("title", entity.OpEq, "go"), `"e"."title" = ?`, []any{"go"}},
		{"ne", filter("views", entity.OpNe, int64(1)), `"e"."views" <> ?`, []any{int64(1)}},
		{"gte on system field", filter(entity.FieldCreatedAt, entity.OpGte, "t"), `"e"."created_at" >= ?`, []any{"t"}},
		{"in", filter(entity.FieldID, entity.OpIn, "a", "b"), `"e"."id" IN (?,?)`, []any{"a", "b"}},
		{"not in", filter("views", entity.OpNotIn, int64(1), int64(2)), `"e"."views" NOT IN (?,?)`, []any{int64(1), int64(2)}},
		{"contains escapes wildcards", filter("title", entity.OpContains, `50%_off\`), `"e"."title" LIKE ?`, []any{`%50\%\_off\\%`}},
		{"containsi", filter("title", entity.OpContainsi, "Go"), `"e"."title" ILIKE ?`, []any{"%Go%"}},
		{"not contains", filter("title", entity.OpNotContains, "x"), `"e"."title" NOT LIKE ?`, []any{"%x%"}},
		{"starts with", filter("title", entity.OpStartsWith, "go"), `"e"."title" LIKE ?`, []any{"go%"}},
		{"ends with", filter("title", entity.OpEndsWith, "go"), `"e"."title" LIKE ?`, []any{"%go"}},
		{"null true", filter("views", entity.OpNull, true), `"e"."views" IS NULL`, nil},
		{"null false", filter("views", entity.OpNull, false), `"e"."views" IS NOT NULL`, nil},
		{"not null true", filter("views", entity.OpNotNull, true), `"e"."views" IS NOT NULL`, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cond, err := condition(&tc.filter)
			require.NoError(t, err)

			sql, args := toSQL(t, cond)
			assert.Equal(t, tc.wantSQL, sql)
			assert.Equal(t, tc.wantArgs, args)
		})
	}
}

func TestConditionQuotesInjectedFieldNames(t *testing.T) {
	t.Parallel()

	cond, err := condition(&entity.Filter{Field: `x" = 1 OR "1`, Operator: entity.OpEq, Values: []any{1}})
	require.NoError(t, err)

	sql, _ := toSQL(t, cond)
	assert.Equal(t, `"e"."x"" = 1 OR ""1" = ?`, sql)
}

func TestOrderBy(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{`"e"."created_at" ASC NULLS LAST`, `"e"."id" ASC`}, orderBy(nil))
	assert.Equal(t,
		[]string{`"e"."views" DESC NULLS LAST`, `"e"."title" ASC NULLS LAST`, `"e"."id" ASC`},
		orderBy([]entity.Sort{{Field: "views", Desc: true}, {Field: "title"}}))
	assert.Equal(t, []string{`"e"."id" DESC NULLS LAST`}, orderBy([]entity.Sort{{Field: entity.FieldID, Desc: true}}))
}

func TestSelectEntries(t *testing.T) {
	t.Parallel()

	t.Run("without media", func(t *testing.T) {
		t.Parallel()

		ct := entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{{Name: "title"}}}

		sql, _ := toSQL(t, selectEntries(sq.StatementBuilder, ct, `"content_articles" AS "e"`))

		assert.Equal(t, `SELECT "e"."id"::text AS "id", "e"."created_at" AS "createdAt", "e"."updated_at" AS "updatedAt",`+
			` "e"."title" FROM "content_articles" AS "e"`, sql)
	})

	t.Run("media field is joined and nested", func(t *testing.T) {
		t.Parallel()

		ct := entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
			{Name: "title"},
			{Name: "cover", Type: entity.AttributeTypeMedia},
		}}

		sql, _ := toSQL(t, selectEntries(sq.StatementBuilder, ct, `"e"`))

		assert.Contains(t, sql, `CASE WHEN "m_cover".id IS NULL THEN NULL ELSE json_build_object(`+
			`'id', "m_cover".id, 'name', "m_cover".name, 'alternativeText', "m_cover".alternative_text, `+
			`'url', "m_cover".url, 'mime', "m_cover".mime, 'ext', "m_cover".ext, 'size', "m_cover".size) END AS "cover"`)
		assert.Contains(t, sql, `FROM "e" LEFT JOIN upload_files AS "m_cover" ON "m_cover".id = "e"."cover"`)
	})
}
