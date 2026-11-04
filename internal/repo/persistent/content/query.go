package content

import (
	"encoding/json"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/lucimux/arya/internal/entity"
)

// _entryAlias names the content table in reads. Media fields join upload_files, which has its own
// id and timestamps, so every column in a read is qualified with it.
const _entryAlias = `"e"`

// column returns the quoted column for a field, unqualified, as INSERT and UPDATE need it. System
// fields are camelCase in the API and snake_case in the table; attributes are the same in both.
func column(field string) string {
	switch field {
	case entity.FieldCreatedAt:
		return `"created_at"`
	case entity.FieldUpdatedAt:
		return `"updated_at"`
	default:
		return pgx.Identifier{field}.Sanitize()
	}
}

// qualified returns the column for a field qualified with _entryAlias, for reads.
func qualified(field string) string {
	return _entryAlias + "." + column(field)
}

// mediaAlias is the upload_files alias joined for one media field.
func mediaAlias(field string) string {
	return pgx.Identifier{"m_" + field}.Sanitize()
}

// mediaObject renders the joined file as a JSON object — or null when the field is empty or its
// file was deleted — so the API returns the file nested under the field instead of a bare id.
func mediaObject(field string) string {
	m := mediaAlias(field)

	return "CASE WHEN " + m + ".id IS NULL THEN NULL ELSE json_build_object(" +
		"'id', " + m + ".id, 'name', " + m + ".name, 'alternativeText', " + m + ".alternative_text, " +
		"'url', " + m + ".url, 'mime', " + m + ".mime, 'ext', " + m + ".ext, 'size', " + m + ".size) END"
}

// selectColumns lists every field, aliased to its API name. The id is cast to text because pgx
// scans a uuid into a [16]byte, which would serialize as an array of numbers.
func selectColumns(ct entity.ContentType) []string {
	cols := []string{
		qualified(entity.FieldID) + "::text AS " + pgx.Identifier{entity.FieldID}.Sanitize(),
		qualified(entity.FieldCreatedAt) + " AS " + pgx.Identifier{entity.FieldCreatedAt}.Sanitize(),
		qualified(entity.FieldUpdatedAt) + " AS " + pgx.Identifier{entity.FieldUpdatedAt}.Sanitize(),
	}

	for _, a := range ct.Attributes {
		if a.Type == entity.AttributeTypeMedia {
			cols = append(cols, mediaObject(a.Name)+" AS "+column(a.Name))

			continue
		}

		cols = append(cols, qualified(a.Name))
	}

	return cols
}

// selectEntries reads entries from source, which must be named _entryAlias: the content table
// ("table AS e") or a CTE called e. Every media field adds a LEFT JOIN to upload_files.
func selectEntries(b sq.StatementBuilderType, ct entity.ContentType, source string) sq.SelectBuilder {
	q := b.Select(selectColumns(ct)...).From(source)

	for _, a := range ct.Attributes {
		if a.Type == entity.AttributeTypeMedia {
			m := mediaAlias(a.Name)
			q = q.LeftJoin("upload_files AS " + m + " ON " + m + ".id = " + qualified(a.Name))
		}
	}

	return q
}

func columnValues(ct entity.ContentType, data entity.Entry) (map[string]any, error) {
	values := make(map[string]any, len(data))
	for field, v := range data {
		col := column(field)
		if a, ok := ct.Attribute(field); ok && a.Type == entity.AttributeTypeComponent {
			if v != nil {
				b, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("columnValues - json.Marshal: %w", err)
				}

				values[col] = sq.Expr("?::jsonb", string(b))

				continue
			}
		}

		values[col] = v
	}

	return values, nil
}

// orderBy defaults to creation order and always ends with the id, so pages are stable when the
// requested sort has ties. Empty values sort last in both directions (Postgres puts them first
// on DESC by default).
func orderBy(sorts []entity.Sort) []string {
	if len(sorts) == 0 {
		sorts = []entity.Sort{{Field: entity.FieldCreatedAt}}
	}

	clauses := make([]string, 0, len(sorts)+1)
	hasID := false

	for _, s := range sorts {
		dir := " ASC NULLS LAST"
		if s.Desc {
			dir = " DESC NULLS LAST"
		}

		clauses = append(clauses, qualified(s.Field)+dir)
		hasID = hasID || s.Field == entity.FieldID
	}

	if !hasID {
		clauses = append(clauses, qualified(entity.FieldID)+" ASC")
	}

	return clauses
}

func whereClause(filters []entity.Filter) (sq.And, error) {
	where := make(sq.And, 0, len(filters))

	for i := range filters {
		cond, err := condition(&filters[i])
		if err != nil {
			return nil, err
		}

		where = append(where, cond)
	}

	return where, nil
}

// condition translates one resolved filter. Values are always bound as parameters.
func condition(f *entity.Filter) (sq.Sqlizer, error) {
	col := qualified(f.Field)

	if len(f.Values) == 0 {
		return nil, fmt.Errorf("%w: filter %q has no value", entity.ErrInvalidQuery, f.Field)
	}

	switch f.Operator {
	case entity.OpIn:
		return sq.Eq{col: f.Values}, nil
	case entity.OpNotIn:
		return sq.NotEq{col: f.Values}, nil
	case entity.OpNull, entity.OpNotNull:
		return nullCondition(col, f.Operator, f.Values[0])
	case entity.OpContains, entity.OpNotContains, entity.OpContainsi, entity.OpNotContainsi,
		entity.OpStartsWith, entity.OpEndsWith:
		return likeCondition(col, f.Operator, f.Values[0])
	case entity.OpEq, entity.OpNe, entity.OpLt, entity.OpLte, entity.OpGt, entity.OpGte:
		return comparison(col, f.Operator, f.Values[0])
	default:
		return nil, fmt.Errorf("%w: unsupported operator %s", entity.ErrInvalidQuery, f.Operator)
	}
}

func comparison(col string, op entity.FilterOperator, v any) (sq.Sqlizer, error) {
	conds := map[entity.FilterOperator]sq.Sqlizer{
		entity.OpEq:  sq.Eq{col: v},
		entity.OpNe:  sq.NotEq{col: v},
		entity.OpLt:  sq.Lt{col: v},
		entity.OpLte: sq.LtOrEq{col: v},
		entity.OpGt:  sq.Gt{col: v},
		entity.OpGte: sq.GtOrEq{col: v},
	}

	cond, ok := conds[op]
	if !ok {
		return nil, fmt.Errorf("%w: %s is not a comparison operator", entity.ErrInvalidQuery, op)
	}

	return cond, nil
}

func nullCondition(col string, op entity.FilterOperator, v any) (sq.Sqlizer, error) {
	want, ok := v.(bool)
	if !ok {
		return nil, fmt.Errorf("%w: %s takes a boolean", entity.ErrInvalidQuery, op)
	}

	// $null=true and $notNull=false both mean IS NULL.
	if want == (op == entity.OpNull) {
		return sq.Eq{col: nil}, nil
	}

	return sq.NotEq{col: nil}, nil
}

func likeCondition(col string, op entity.FilterOperator, v any) (sq.Sqlizer, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("%w: %s takes a string", entity.ErrInvalidQuery, op)
	}

	s = escapeLike(s)

	conds := map[entity.FilterOperator]sq.Sqlizer{
		entity.OpContains:     sq.Like{col: "%" + s + "%"},
		entity.OpNotContains:  sq.NotLike{col: "%" + s + "%"},
		entity.OpContainsi:    sq.ILike{col: "%" + s + "%"},
		entity.OpNotContainsi: sq.NotILike{col: "%" + s + "%"},
		entity.OpStartsWith:   sq.Like{col: s + "%"},
		entity.OpEndsWith:     sq.Like{col: "%" + s},
	}

	cond, ok := conds[op]
	if !ok {
		return nil, fmt.Errorf("%w: %s is not a text operator", entity.ErrInvalidQuery, op)
	}

	return cond, nil
}

// escapeLike makes LIKE wildcards in user input match literally. Backslash is Postgres' default
// LIKE escape character.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
