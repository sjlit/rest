package query

// ---- Source ----

type Source interface {
	isSource()
}

type TableSource string

func (TableSource) isSource() {}

type SubquerySource struct {
	Subquery Subquery
}

func (SubquerySource) isSource() {}

// ---- Expr ----

type Expr interface {
	isExpr()
}

type FieldExpr struct {
	Name string
}

func (FieldExpr) isExpr() {}

type AliasedExpr struct {
	Expr  Expr
	Alias string
}

func (AliasedExpr) isExpr() {}

type AggregateExpr struct {
	Field    string
	Function string
	Alias    string
}

func (AggregateExpr) isExpr() {}

func (a AggregateExpr) As(alias string) Expr {
	a.Alias = alias
	return a
}

type RawExpr struct {
	SQL  string
	Args []any
}

func (RawExpr) isExpr() {}

// ---- Helpers ----

func Field(name string) Expr         { return FieldExpr{Name: name} }
func Count(field string) AggregateExpr { return AggregateExpr{Field: field, Function: "COUNT"} }
func Sum(field string) AggregateExpr   { return AggregateExpr{Field: field, Function: "SUM"} }
func Avg(field string) AggregateExpr   { return AggregateExpr{Field: field, Function: "AVG"} }
func Max(field string) AggregateExpr   { return AggregateExpr{Field: field, Function: "MAX"} }
func Min(field string) AggregateExpr   { return AggregateExpr{Field: field, Function: "MIN"} }
func Raw(sql string, args ...any) Expr { return RawExpr{SQL: sql, Args: args} }

// ---- Clause ----

type SubqueryClause struct {
	Op       Operator
	Subquery Subquery
}

func (SubqueryClause) isClause() {}

// ---- Supporting Types ----

type Subquery struct {
	Spec  QuerySpec
	Alias string
}

func NewSubquery(fn func(*Builder)) Subquery {
	sub := NewBuilder()
	fn(sub)
	return Subquery{Spec: sub.spec}
}

// ---- QuerySpec ----

type QuerySpec struct {
	Selects []Expr
	Source  Source
	Joins   []Join
	Where   []Clause
	GroupBy []string
	Having  []Clause
	OrderBy []Order
	Limit   int
	Offset  int
}

func (s QuerySpec) Clone() QuerySpec {
	cloned := QuerySpec{
		Source: s.Source,
		Limit:  s.Limit,
		Offset: s.Offset,
	}
	if len(s.Selects) > 0 {
		cloned.Selects = make([]Expr, len(s.Selects))
		copy(cloned.Selects, s.Selects)
	}
	if len(s.Joins) > 0 {
		cloned.Joins = make([]Join, len(s.Joins))
		copy(cloned.Joins, s.Joins)
	}
	if len(s.Where) > 0 {
		cloned.Where = make([]Clause, len(s.Where))
		copy(cloned.Where, s.Where)
	}
	if len(s.GroupBy) > 0 {
		cloned.GroupBy = make([]string, len(s.GroupBy))
		copy(cloned.GroupBy, s.GroupBy)
	}
	if len(s.Having) > 0 {
		cloned.Having = make([]Clause, len(s.Having))
		copy(cloned.Having, s.Having)
	}
	if len(s.OrderBy) > 0 {
		cloned.OrderBy = make([]Order, len(s.OrderBy))
		copy(cloned.OrderBy, s.OrderBy)
	}
	return cloned
}
