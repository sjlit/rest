package query

import "strings"

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
	spec QuerySpec
}

func (b *Builder) Spec() QuerySpec {
	return b.spec
}

func (b *Builder) Where(field string, op Operator, value any) *Builder {
	b.spec.Where = append(b.spec.Where, Condition{Field: field, Op: op, Value: value})
	return b
}

func (b *Builder) OrWhere(field string, op Operator, value any) *Builder {
	b.spec.Where = append(b.spec.Where, OrGroup{
		Clauses: []Clause{Condition{Field: field, Op: op, Value: value}},
	})
	return b
}

func (b *Builder) WhereGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.spec.Where) > 0 {
		b.spec.Where = append(b.spec.Where, AndGroup{Clauses: sub.spec.Where})
	}
	return b
}

func (b *Builder) OrWhereGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.spec.Where) > 0 {
		b.spec.Where = append(b.spec.Where, OrGroup{Clauses: sub.spec.Where})
	}
	return b
}

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
	b.spec.Where = append(b.spec.Where, Condition{Field: field, Op: OpLike, Value: pattern})
	return b
}

func (b *Builder) From(table string) *Builder {
	b.spec.Source = TableSource(table)
	return b
}

func (b *Builder) FromSubquery(alias string, fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	b.spec.Source = SubquerySource{Subquery: Subquery{Spec: sub.spec, Alias: alias}}
	return b
}

func (b *Builder) Join(direction, table, on string, args ...any) *Builder {
	b.spec.Joins = append(b.spec.Joins, Join{
		Direction: direction,
		Table:     table,
		On:        on,
		Args:      args,
	})
	return b
}

func (b *Builder) LeftJoin(table, on string, args ...any) *Builder {
	return b.Join("LEFT", table, on, args...)
}

func (b *Builder) InnerJoin(table, on string, args ...any) *Builder {
	return b.Join("INNER", table, on, args...)
}

func (b *Builder) Select(exprs ...Expr) *Builder {
	b.spec.Selects = append(b.spec.Selects, exprs...)
	return b
}

func (b *Builder) OrderBy(field, direction string) *Builder {
	if strings.ToUpper(direction) == "DESC" {
		direction = "DESC"
	} else {
		direction = "ASC"
	}
	b.spec.OrderBy = append(b.spec.OrderBy, Order{Field: field, Direction: direction})
	return b
}

func (b *Builder) GroupBy(fields ...string) *Builder {
	b.spec.GroupBy = append(b.spec.GroupBy, fields...)
	return b
}

func (b *Builder) Limit(n int) *Builder {
	b.spec.Limit = n
	return b
}

func (b *Builder) Offset(n int) *Builder {
	b.spec.Offset = n
	return b
}

func (b *Builder) Having(field string, op Operator, value any) *Builder {
	b.spec.Having = append(b.spec.Having, Condition{Field: field, Op: op, Value: value})
	return b
}

func (b *Builder) OrHaving(field string, op Operator, value any) *Builder {
	b.spec.Having = append(b.spec.Having, OrGroup{
		Clauses: []Clause{Condition{Field: field, Op: op, Value: value}},
	})
	return b
}

func (b *Builder) HavingGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.spec.Having) > 0 {
		b.spec.Having = append(b.spec.Having, AndGroup{Clauses: sub.spec.Having})
	}
	return b
}

func (b *Builder) OrHavingGroup(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	if len(sub.spec.Having) > 0 {
		b.spec.Having = append(b.spec.Having, OrGroup{Clauses: sub.spec.Having})
	}
	return b
}

func (b *Builder) WhereExists(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	b.spec.Where = append(b.spec.Where, SubqueryClause{
		Op:       OpExists,
		Subquery: Subquery{Spec: sub.spec},
	})
	return b
}

func (b *Builder) WhereNotExists(fn func(*Builder)) *Builder {
	sub := NewBuilder()
	fn(sub)
	b.spec.Where = append(b.spec.Where, SubqueryClause{
		Op:       OpNotExists,
		Subquery: Subquery{Spec: sub.spec},
	})
	return b
}

func (b *Builder) Clone() *Builder {
	return &Builder{spec: b.spec.Clone()}
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
