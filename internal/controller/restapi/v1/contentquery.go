package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/entity"
)

// queryParams returns every query parameter in order, repeats included (filters[id][$in]=a&...=b).
func queryParams(ctx *fiber.Ctx) [][2]string {
	var params [][2]string

	for k, v := range ctx.Request().URI().QueryArgs().All() {
		params = append(params, [2]string{string(k), string(v)})
	}

	return params
}

// _maxFilterKeyParts is field, operator and index: filters[field][$op][index].
const _maxFilterKeyParts = 3

// parseContentQuery reads Strapi-style query parameters:
//
//	pagination[page]=1&pagination[pageSize]=10
//	sort=createdAt:desc,title:asc   (or sort[0]=createdAt:desc&sort[1]=title:asc)
//	filters[title][$contains]=go    (filters[title]=go is shorthand for $eq)
//	filters[id][$in][0]=a&filters[id][$in][1]=b
//
// It only checks syntax. Whether a field exists or an operator fits its type is decided by
// entity.ContentType.ResolveQuery. Other parameters are ignored. Errors wrap entity.ErrInvalidQuery.
func parseContentQuery(params [][2]string) (entity.ContentQuery, error) {
	q := entity.ContentQuery{Page: 1, PageSize: entity.DefaultPageSize}
	filters := newFilterSet()

	for _, kv := range params {
		key, value := kv[0], kv[1]

		var err error

		switch {
		case strings.HasPrefix(key, "pagination["):
			err = parsePagination(&q, key, value)
		case key == "sort" || strings.HasPrefix(key, "sort["):
			q.Sort, err = appendSorts(q.Sort, value)
		case strings.HasPrefix(key, "filters["):
			err = filters.add(key, value)
		}

		if err != nil {
			return entity.ContentQuery{}, err
		}
	}

	q.Filters = filters.list()

	return q, nil
}

func parsePagination(q *entity.ContentQuery, key, value string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%w: %s=%q is not a number", entity.ErrInvalidQuery, key, value)
	}

	switch key {
	case "pagination[page]":
		q.Page = n
	case "pagination[pageSize]":
		q.PageSize = n
	default:
		return fmt.Errorf("%w: unsupported parameter %s", entity.ErrInvalidQuery, key)
	}

	return nil
}

// appendSorts parses "field[:asc|:desc]" items separated by commas.
func appendSorts(sorts []entity.Sort, value string) ([]entity.Sort, error) {
	for item := range strings.SplitSeq(value, ",") {
		field, dir, _ := strings.Cut(strings.TrimSpace(item), ":")
		if field == "" {
			return nil, fmt.Errorf("%w: empty sort field in %q", entity.ErrInvalidQuery, value)
		}

		switch strings.ToLower(dir) {
		case "", "asc":
			sorts = append(sorts, entity.Sort{Field: field})
		case "desc":
			sorts = append(sorts, entity.Sort{Field: field, Desc: true})
		default:
			return nil, fmt.Errorf("%w: sort direction %q", entity.ErrInvalidQuery, dir)
		}
	}

	return sorts, nil
}

// filterSet groups values by field and operator, keeping the order in which filters first appear.
type filterSet struct {
	order   []string
	filters map[string]*entity.Filter
}

func newFilterSet() *filterSet {
	return &filterSet{filters: make(map[string]*entity.Filter)}
}

func (s *filterSet) add(key, value string) error {
	field, op, err := parseFilterKey(key)
	if err != nil {
		return err
	}

	id := field + "\x00" + string(op)

	f, ok := s.filters[id]
	if !ok {
		f = &entity.Filter{Field: field, Operator: op}
		s.filters[id] = f
		s.order = append(s.order, id)
	}

	f.Raw = append(f.Raw, value)

	return nil
}

func (s *filterSet) list() []entity.Filter {
	out := make([]entity.Filter, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, *s.filters[id])
	}

	return out
}

// parseFilterKey splits filters[field], filters[field][$op] and filters[field][$op][index].
func parseFilterKey(key string) (string, entity.FilterOperator, error) {
	inner, ok := strings.CutPrefix(key, "filters[")
	if ok {
		inner, ok = strings.CutSuffix(inner, "]")
	}

	parts := strings.Split(inner, "][")
	if !ok || parts[0] == "" || len(parts) > _maxFilterKeyParts {
		return "", "", fmt.Errorf("%w: malformed filter %s", entity.ErrInvalidQuery, key)
	}

	op := entity.OpEq
	if len(parts) > 1 {
		op = entity.FilterOperator(parts[1])
	}

	if len(parts) == _maxFilterKeyParts {
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return "", "", fmt.Errorf("%w: malformed filter index in %s", entity.ErrInvalidQuery, key)
		}
	}

	return parts[0], op, nil
}
