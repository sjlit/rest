package query

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// ---- Validation ----

var safeFieldName = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

func validateFieldName(field string) error {
	if field == "" {
		return fmt.Errorf("field name cannot be empty")
	}
	if !safeFieldName.MatchString(field) {
		return fmt.Errorf("invalid field name: %s", field)
	}
	return nil
}

func validateTableName(table string) error {
	if table == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if !safeFieldName.MatchString(table) {
		return fmt.Errorf("invalid table name: %s", table)
	}
	return nil
}

// ---- Operators ----

type Operator string

const (
	OpEq        Operator = "="
	OpNe        Operator = "<>"
	OpGt        Operator = ">"
	OpLt        Operator = "<"
	OpGte       Operator = ">="
	OpLte       Operator = "<="
	OpLike      Operator = "LIKE"
	OpIn        Operator = "IN"
	OpBetween   Operator = "BETWEEN"
	OpIsNull    Operator = "IS NULL"
	OpIsNotNull Operator = "IS NOT NULL"
	OpExists    Operator = "EXISTS"
	OpNotExists Operator = "NOT EXISTS"
)

// ParseOperator maps common operator strings to typed Operator constants.
func ParseOperator(s string) Operator {
	switch strings.ToLower(s) {
	case "eq", "=":
		return OpEq
	case "ne", "<>", "!=":
		return OpNe
	case "gt", ">":
		return OpGt
	case "lt", "<":
		return OpLt
	case "ge", "gte", ">=":
		return OpGte
	case "le", "lte", "<=":
		return OpLte
	case "like":
		return OpLike
	case "in":
		return OpIn
	case "between":
		return OpBetween
	case "isnull", "is_null", "is null":
		return OpIsNull
	case "isnotnull", "is_not_null", "is not null":
		return OpIsNotNull
	default:
		return Operator(s)
	}
}

// ---- Types ----

type LikeMode int

const (
	LikeContains LikeMode = iota
	LikePrefix
	LikeSuffix
)

// Clause is the interface for all query conditions.
type Clause interface {
	isClause()
}

// Condition represents a single field-operator-value condition.
type Condition struct {
	Field string
	Op    Operator
	Value any
}

func (Condition) isClause() {}

// AndGroup groups clauses with AND logic.
type AndGroup struct {
	Clauses []Clause
}

func (AndGroup) isClause() {}

// OrGroup groups clauses with OR logic.
type OrGroup struct {
	Clauses []Clause
}

func (OrGroup) isClause() {}

// Join represents a table join.
type Join struct {
	Direction string
	Table     string
	On        string
	Args      []any
}

// Order represents a single ORDER BY expression.
type Order struct {
	Field     string
	Direction string
}

// ---- Query ----

// Query binds a Builder to a GORM DB instance and executes the query.
type Query struct {
	db      *gorm.DB
	model   any
	builder *Builder
}

// New creates a new Query.
func New(db *gorm.DB, model any, builder *Builder) *Query {
	return &Query{
		db:      db,
		model:   model,
		builder: builder,
	}
}

// Builder returns the underlying Builder.
func (q *Query) Builder() *Builder {
	return q.builder
}

// ---- Execution ----

// Count returns the total number of matching rows.
func (q *Query) Count(ctx context.Context) (int64, error) {
	db, err := q.compile(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Count(&count).Error
	return count, err
}

// One fetches a single result using Take (no implicit ordering).
func (q *Query) One(ctx context.Context, dest any) error {
	db, err := q.compile(ctx)
	if err != nil {
		return err
	}
	return db.Take(dest).Error
}

// All fetches all matching results.
func (q *Query) All(ctx context.Context, dest any) error {
	db, err := q.compile(ctx)
	if err != nil {
		return err
	}
	return db.Find(dest).Error
}

// Page executes a paginated query and returns total count and results.
// page starts from 1.
func (q *Query) Page(ctx context.Context, page, size int, dest any) (int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	db, err := q.compile(ctx)
	if err != nil {
		return 0, err
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, err
	}

	err = db.Offset((page - 1) * size).Limit(size).Find(dest).Error
	return total, err
}

// ---- Compilation ----

func (q *Query) compile(ctx context.Context) (*gorm.DB, error) {
	db := q.db.WithContext(ctx)

	switch src := q.builder.spec.Source.(type) {
	case TableSource:
		db = db.Table(string(src))
	default:
		if q.model != nil {
			db = db.Model(q.model)
		}
	}

	for _, j := range q.builder.spec.Joins {
		joinSQL := fmt.Sprintf("%s JOIN %s ON %s", j.Direction, j.Table, j.On)
		db = db.Joins(joinSQL, j.Args...)
	}

	db, err := compileClauses(db, q.builder.spec.Where)
	if err != nil {
		return nil, err
	}

	if len(q.builder.spec.Selects) > 0 {
		var selects []string
		for _, expr := range q.builder.spec.Selects {
			selects = append(selects, compileExpr(expr))
		}
		db = db.Select(selects)
	}
	if len(q.builder.spec.OrderBy) > 0 {
		for _, o := range q.builder.spec.OrderBy {
			db = db.Order(fmt.Sprintf("%s %s", o.Field, o.Direction))
		}
	}
	if len(q.builder.spec.GroupBy) > 0 {
		for _, g := range q.builder.spec.GroupBy {
			db = db.Group(g)
		}
	}
	if q.builder.spec.Offset > 0 {
		db = db.Offset(q.builder.spec.Offset)
	}
	if q.builder.spec.Limit > 0 {
		db = db.Limit(q.builder.spec.Limit)
	}

	return db, nil
}

func compileExpr(expr Expr) string {
	switch e := expr.(type) {
	case FieldExpr:
		return e.Name
	case AliasedExpr:
		return fmt.Sprintf("%s AS %s", compileExpr(e.Expr), e.Alias)
	case AggregateExpr:
		if e.Alias != "" {
			return fmt.Sprintf("%s(%s) AS %s", e.Function, e.Field, e.Alias)
		}
		return fmt.Sprintf("%s(%s)", e.Function, e.Field)
	case RawExpr:
		return e.SQL
	default:
		return ""
	}
}

func compileClauses(db *gorm.DB, clauses []Clause) (*gorm.DB, error) {
	if len(clauses) == 0 {
		return db, nil
	}

	// Compile first clause
	var err error
	db, err = compileClauseAsAnd(db, clauses[0])
	if err != nil {
		return nil, err
	}

	// Compile remaining clauses
	for _, clause := range clauses[1:] {
		switch c := clause.(type) {
		case Condition, AndGroup:
			db, err = compileClauseAsAnd(db, c)
		case OrGroup:
			db, err = compileClauseAsOr(db, c)
		}
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func compileClauseAsAnd(db *gorm.DB, clause Clause) (*gorm.DB, error) {
	switch c := clause.(type) {
	case Condition:
		return compileCondition(db, c)
	case AndGroup:
		sub, err := compileClauses(db.Session(&gorm.Session{NewDB: true}), c.Clauses)
		if err != nil {
			return nil, err
		}
		return db.Where(sub), nil
	}
	return nil, fmt.Errorf("unexpected clause type in AND context: %T", clause)
}

func compileClauseAsOr(db *gorm.DB, clause OrGroup) (*gorm.DB, error) {
	sub, err := compileClauses(db.Session(&gorm.Session{NewDB: true}), clause.Clauses)
	if err != nil {
		return nil, err
	}
	return db.Or(sub), nil
}

// conditionCompiler is a function that compiles a single Condition into a GORM DB.
type conditionCompiler func(db *gorm.DB, field string, value any) (*gorm.DB, error)

var compilers = map[Operator]conditionCompiler{
	OpEq:        compileEq,
	OpNe:        compileNe,
	OpGt:        compileSimple(">"),
	OpLt:        compileSimple("<"),
	OpGte:       compileSimple(">="),
	OpLte:       compileSimple("<="),
	OpLike:      compileLike,
	OpIn:        compileIn,
	OpBetween:   compileBetween,
	OpIsNull:    compileIsNull,
	OpIsNotNull: compileIsNotNull,
}

func compileCondition(db *gorm.DB, c Condition) (*gorm.DB, error) {
	compiler, ok := compilers[c.Op]
	if !ok {
		return nil, fmt.Errorf("unsupported operator: %s", c.Op)
	}
	return compiler(db, c.Field, c.Value)
}

func compileEq(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	if value == nil {
		return db.Where(fmt.Sprintf("%s IS NULL", field)), nil
	}
	return db.Where(fmt.Sprintf("%s = ?", field), value), nil
}

func compileNe(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	if value == nil {
		return db.Where(fmt.Sprintf("%s IS NOT NULL", field)), nil
	}
	return db.Where(fmt.Sprintf("%s <> ?", field), value), nil
}

func compileSimple(op string) conditionCompiler {
	return func(db *gorm.DB, field string, value any) (*gorm.DB, error) {
		return db.Where(fmt.Sprintf("%s %s ?", field, op), value), nil
	}
}

func compileLike(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s LIKE ? ESCAPE '\\'", field), value), nil
}

func compileIn(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IN ?", field), value), nil
}

func compileBetween(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice || v.Len() != 2 {
		return nil, fmt.Errorf("BETWEEN operator requires a slice of length 2")
	}
	return db.Where(
		fmt.Sprintf("%s BETWEEN ? AND ?", field),
		v.Index(0).Interface(),
		v.Index(1).Interface(),
	), nil
}

func compileIsNull(db *gorm.DB, field string, _ any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IS NULL", field)), nil
}

func compileIsNotNull(db *gorm.DB, field string, _ any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IS NOT NULL", field)), nil
}
