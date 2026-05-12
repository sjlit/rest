package query

import (
	"fmt"
	"regexp"
	"strings"
)

// NewBuilder creates a new query Builder.
func NewBuilder() *Builder {
	return &Builder{
		clauses: make([]Clause, 0),
		selects: make([]string, 0),
		joins:   make([]Join, 0),
		orderBy: make([]Order, 0),
		groupBy: make([]string, 0),
	}
}

// Builder constructs a query declaratively using a fluent API.
type Builder struct {
	clauses []Clause
	selects []string
	table   string
	joins   []Join
	orderBy []Order
	groupBy []string
	limit   int
	offset  int
}

// ---- Clauses ----

// Where appends an AND condition.
func (b *Builder) Where(field string, op Operator, value any) *Builder {
	b.clauses = append(b.clauses, Condition{Field: field, Op: op, Value: value})
	return b
}

// OrWhere appends an OR condition.
// The OR applies at the current nesting level.
func (b *Builder) OrWhere(field string, op Operator, value any) *Builder {
	b.clauses = append(b.clauses, OrGroup{
		Clauses: []Clause{Condition{Field: field, Op: op, Value: value}},
	})
	return b
}

// WhereGroup appends a nested AND group.
func (b *Builder) WhereGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.clauses) > 0 {
		b.clauses = append(b.clauses, AndGroup{Clauses: sub.clauses})
	}
	return b
}

// OrWhereGroup appends a nested OR group.
func (b *Builder) OrWhereGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.clauses) > 0 {
		b.clauses = append(b.clauses, OrGroup{Clauses: sub.clauses})
	}
	return b
}

// Like appends a LIKE condition with controlled pattern mode.
func (b *Builder) Like(field string, value string, mode LikeMode) *Builder {
	value = escapeLikePattern(value)
	var pattern string
	switch mode {
	case LikePrefix:
		pattern = value + "%"
	case LikeSuffix:
		pattern = "%" + value
	default:
		pattern = "%" + value + "%"
	}
	b.clauses = append(b.clauses, Condition{Field: field, Op: OpLike, Value: pattern})
	return b
}

// ---- Table & Joins ----

// From sets the target table name.
func (b *Builder) From(table string) *Builder {
	b.table = table
	return b
}

// Join appends a JOIN clause.
// The 'on' argument is a raw SQL fragment; values should use ? placeholders passed via args.
func (b *Builder) Join(direction, table, on string, args ...any) *Builder {
	b.joins = append(b.joins, Join{
		Direction: direction,
		Table:     table,
		On:        on,
		Args:      args,
	})
	return b
}

// LeftJoin appends a LEFT JOIN.
func (b *Builder) LeftJoin(table, on string, args ...any) *Builder {
	return b.Join("LEFT", table, on, args...)
}

// InnerJoin appends an INNER JOIN.
func (b *Builder) InnerJoin(table, on string, args ...any) *Builder {
	return b.Join("INNER", table, on, args...)
}

// ---- Projection ----

// Select appends fields to the SELECT clause.
func (b *Builder) Select(fields ...string) *Builder {
	b.selects = append(b.selects, fields...)
	return b
}

// ---- Ordering & Grouping ----

// OrderBy appends an ORDER BY expression.
func (b *Builder) OrderBy(field, direction string) *Builder {
	if strings.ToUpper(direction) == "DESC" {
		direction = "DESC"
	} else {
		direction = "ASC"
	}
	b.orderBy = append(b.orderBy, Order{Field: field, Direction: direction})
	return b
}

// GroupBy appends GROUP BY fields.
func (b *Builder) GroupBy(fields ...string) *Builder {
	b.groupBy = append(b.groupBy, fields...)
	return b
}

// ---- Pagination ----

// Limit sets the LIMIT.
func (b *Builder) Limit(n int) *Builder {
	b.limit = n
	return b
}

// Offset sets the OFFSET.
func (b *Builder) Offset(n int) *Builder {
	b.offset = n
	return b
}

// ---- Validation ----

var safeJoinOn = regexp.MustCompile(`^[a-zA-Z0-9_\.\s\=\<\>\!\?\,]+$`)

func validateJoinOn(on string) error {
	if on == "" {
		return fmt.Errorf("join condition cannot be empty")
	}
	if strings.Contains(on, ";") || strings.Contains(on, "--") || strings.Contains(on, "/*") {
		return fmt.Errorf("invalid join condition: contains forbidden characters")
	}
	if !safeJoinOn.MatchString(on) {
		return fmt.Errorf("invalid join condition")
	}
	return nil
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func (b *Builder) validate() error {
	for _, clause := range b.clauses {
		if err := validateClause(clause); err != nil {
			return err
		}
	}
	for _, s := range b.selects {
		if err := validateFieldName(s); err != nil {
			return err
		}
	}
	for _, j := range b.joins {
		if err := validateTableName(j.Table); err != nil {
			return err
		}
		if err := validateJoinOn(j.On); err != nil {
			return err
		}
	}
	for _, o := range b.orderBy {
		if err := validateFieldName(o.Field); err != nil {
			return err
		}
	}
	for _, g := range b.groupBy {
		if err := validateFieldName(g); err != nil {
			return err
		}
	}
	return nil
}

func validateClause(clause Clause) error {
	switch c := clause.(type) {
	case Condition:
		return validateFieldName(c.Field)
	case AndGroup:
		for _, cl := range c.Clauses {
			if err := validateClause(cl); err != nil {
				return err
			}
		}
	case OrGroup:
		for _, cl := range c.Clauses {
			if err := validateClause(cl); err != nil {
				return err
			}
		}
	}
	return nil
}
