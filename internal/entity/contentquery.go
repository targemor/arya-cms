package entity

import (
	"fmt"
	"slices"
)

// Pagination defaults and limits, as in Strapi.
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// FilterOperator is a Strapi-style filter operator, e.g. "$eq".
type FilterOperator string

// Supported filter operators.
const (
	OpEq           FilterOperator = "$eq"
	OpNe           FilterOperator = "$ne"
	OpLt           FilterOperator = "$lt"
	OpLte          FilterOperator = "$lte"
	OpGt           FilterOperator = "$gt"
	OpGte          FilterOperator = "$gte"
	OpIn           FilterOperator = "$in"
	OpNotIn        FilterOperator = "$notIn"
	OpContains     FilterOperator = "$contains"
	OpNotContains  FilterOperator = "$notContains"
	OpContainsi    FilterOperator = "$containsi"
	OpNotContainsi FilterOperator = "$notContainsi"
	OpStartsWith   FilterOperator = "$startsWith"
	OpEndsWith     FilterOperator = "$endsWith"
	OpNull         FilterOperator = "$null"
	OpNotNull      FilterOperator = "$notNull"
)

// Filter is one condition. Raw holds the query-string values; ResolveQuery fills Values with
// them converted to the field's type. $in and $notIn take any number of values, $null and
// $notNull take a boolean, every other operator exactly one value.
type Filter struct {
	Field    string
	Operator FilterOperator
	Raw      []string
	Values   []any
}

// Sort orders by one field.
type Sort struct {
	Field string
	Desc  bool
}

// ContentQuery selects a page of entries.
type ContentQuery struct {
	Page     int
	PageSize int
	Sort     []Sort
	Filters  []Filter
}

// Pagination describes the page returned for a ContentQuery.
type Pagination struct {
	Page      int `json:"page"      example:"1"`
	PageSize  int `json:"pageSize"  example:"25"`
	PageCount int `json:"pageCount" example:"4"`
	Total     int `json:"total"     example:"87"`
} // @name entity.Pagination

// NewPagination computes the page count for total entries.
func NewPagination(page, pageSize, total int) Pagination {
	return Pagination{Page: page, PageSize: pageSize, PageCount: (total + pageSize - 1) / pageSize, Total: total}
}

// operatorsByType lists which operators make sense for each field type.
func operatorsByType(t FieldType) []FilterOperator {
	equality := []FilterOperator{OpEq, OpNe, OpIn, OpNotIn, OpNull, OpNotNull}
	ordering := []FilterOperator{OpLt, OpLte, OpGt, OpGte}
	text := []FilterOperator{OpContains, OpNotContains, OpContainsi, OpNotContainsi, OpStartsWith, OpEndsWith}

	switch t {
	case FieldTypeString:
		return slices.Concat(equality, ordering, text)
	case FieldTypeInteger, FieldTypeDatetime:
		return slices.Concat(equality, ordering)
	case FieldTypeBoolean:
		return []FilterOperator{OpEq, OpNe, OpNull, OpNotNull}
	case FieldTypeUUID:
		return []FilterOperator{OpEq, OpNe, OpIn, OpNotIn}
	case FieldTypeMedia:
		return equality
	default:
		return nil
	}
}

// ResolveQuery checks q against ct — page bounds, known fields, operators that fit the field's
// type — and returns a copy whose filters carry typed Values. Errors wrap ErrInvalidQuery.
func (ct ContentType) ResolveQuery(q ContentQuery) (ContentQuery, error) {
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > MaxPageSize {
		return ContentQuery{}, fmt.Errorf("%w: page %d, page size %d", ErrInvalidQuery, q.Page, q.PageSize)
	}

	for _, s := range q.Sort {
		if _, ok := ct.FieldType(s.Field); !ok {
			return ContentQuery{}, fmt.Errorf("%w: unknown sort field %q", ErrInvalidQuery, s.Field)
		}
	}

	resolved := q
	resolved.Filters = make([]Filter, 0, len(q.Filters))

	for i := range q.Filters {
		rf, err := ct.resolveFilter(&q.Filters[i])
		if err != nil {
			return ContentQuery{}, err
		}

		resolved.Filters = append(resolved.Filters, rf)
	}

	return resolved, nil
}

func (ct ContentType) resolveFilter(in *Filter) (Filter, error) {
	f := *in

	ft, ok := ct.FieldType(f.Field)
	if !ok {
		return Filter{}, fmt.Errorf("%w: unknown filter field %q", ErrInvalidQuery, f.Field)
	}

	if !slices.Contains(operatorsByType(ft), f.Operator) {
		return Filter{}, fmt.Errorf("%w: operator %s not supported on %s field %q", ErrInvalidQuery, f.Operator, ft, f.Field)
	}

	if err := checkArity(in); err != nil {
		return Filter{}, err
	}

	// The value of $null / $notNull says whether the condition applies: $null=true.
	if f.Operator == OpNull || f.Operator == OpNotNull {
		ft = FieldTypeBoolean
	}

	f.Values = make([]any, 0, len(f.Raw))

	for _, raw := range f.Raw {
		v, err := ft.parseValue(raw)
		if err != nil {
			return Filter{}, fmt.Errorf("filter %q %s: %w", f.Field, f.Operator, err)
		}

		f.Values = append(f.Values, v)
	}

	return f, nil
}

// checkArity: $in and $notIn take one or more values, every other operator exactly one.
func checkArity(f *Filter) error {
	isList := f.Operator == OpIn || f.Operator == OpNotIn
	if isList && len(f.Raw) == 0 {
		return fmt.Errorf("%w: %s on %q needs at least one value", ErrInvalidQuery, f.Operator, f.Field)
	}

	if !isList && len(f.Raw) != 1 {
		return fmt.Errorf("%w: %s on %q takes exactly one value", ErrInvalidQuery, f.Operator, f.Field)
	}

	return nil
}
