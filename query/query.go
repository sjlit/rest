package query

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// ---- Validation (moved to validator.go) ----
// REMOVED: safeFieldName, validateFieldName, validateTableName

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

type Condition struct {
	Field string
	Op    Operator
	Value any
}

func (Condition) isClause() {}

type AndGroup struct{ Clauses []Clause }

func (AndGroup) isClause() {}

type OrGroup struct{ Clauses []Clause }

func (OrGroup) isClause() {}

type Join struct {
	Direction string
	Table     string
	On        string
	Args      []any
}

type Order struct {
	Field     string
	Direction string
}

type Query struct {
	db      *gorm.DB
	model   any
	builder *Builder
}

func New(db *gorm.DB, model any, builder *Builder) *Query {
	return &Query{db: db, model: model, builder: builder}
}

func (q *Query) Builder() *Builder {
	return q.builder
}

func (q *Query) Count(ctx context.Context) (int64, error) {
	spec := q.builder.Spec().Clone()
	spec.Offset = 0
	spec.Limit = 0

	db, err := q.compileWithSpec(ctx, spec)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Count(&count).Error
	return count, err
}

func (q *Query) One(ctx context.Context, dest any) error {
	db, err := q.compile(ctx)
	if err != nil {
		return err
	}
	return db.Take(dest).Error
}

func (q *Query) All(ctx context.Context, dest any) error {
	db, err := q.compile(ctx)
	if err != nil {
		return err
	}
	return db.Find(dest).Error
}

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

func (q *Query) compile(ctx context.Context) (*gorm.DB, error) {
	return q.compileWithSpec(ctx, q.builder.Spec())
}

func (q *Query) compileWithSpec(ctx context.Context, spec QuerySpec) (*gorm.DB, error) {
	if err := Validate(spec); err != nil {
		return nil, err
	}
	db := q.db.WithContext(ctx)
	if q.model != nil {
		db = db.Model(q.model)
	}
	compiler := NewCompiler()
	return compiler.Compile(spec, db)
}
