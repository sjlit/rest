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
