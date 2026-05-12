package query

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

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

type Operator string

const (
	OpEq      Operator = "="
	OpNe      Operator = "<>"
	OpGt      Operator = ">"
	OpLt      Operator = "<"
	OpGte     Operator = ">="
	OpLte     Operator = "<="
	OpLike    Operator = "LIKE"
	OpIn      Operator = "IN"
	OpBetween Operator = "BETWEEN"
	OpIsNull  Operator = "IS NULL"
)

// ParseOperator maps common operator strings to typed Operator constants.
func ParseOperator(s string) Operator {
	switch strings.ToLower(s) {
	case "eq", "=":
		return OpEq
	case "ne", "<>":
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
	default:
		return Operator(s)
	}
}

type Condition struct {
	Field string
	Op    Operator
	Value any
}

type Join struct {
	Table      string
	Direction  string
	Conditions []Condition
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
	return &Query{
		db:      db,
		model:   model,
		builder: builder,
	}
}

func (q *Query) compile(ctx context.Context) (*gorm.DB, error) {
	db := q.db.WithContext(ctx)
	if q.builder.table != "" {
		db = db.Table(q.builder.table)
	} else if q.model != nil {
		db = db.Model(q.model)
	}

	for _, c := range q.builder.conditions {
		if err := validateFieldName(c.Field); err != nil {
			return nil, err
		}
		switch c.Op {
		case OpEq:
			if c.Value == nil {
				db = db.Where(fmt.Sprintf("%s IS NULL", c.Field))
			} else {
				db = db.Where(fmt.Sprintf("%s = ?", c.Field), c.Value)
			}
		case OpNe:
			db = db.Where(fmt.Sprintf("%s <> ?", c.Field), c.Value)
		case OpGt:
			db = db.Where(fmt.Sprintf("%s > ?", c.Field), c.Value)
		case OpLt:
			db = db.Where(fmt.Sprintf("%s < ?", c.Field), c.Value)
		case OpGte:
			db = db.Where(fmt.Sprintf("%s >= ?", c.Field), c.Value)
		case OpLte:
			db = db.Where(fmt.Sprintf("%s <= ?", c.Field), c.Value)
		case OpLike:
			db = db.Where(fmt.Sprintf("%s LIKE ?", c.Field), fmt.Sprintf("%%%v%%", c.Value))
		case OpIn:
			db = db.Where(fmt.Sprintf("%s IN ?", c.Field), c.Value)
		case OpBetween:
			vals := reflect.ValueOf(c.Value)
			if vals.Kind() == reflect.Slice && vals.Len() == 2 {
				db = db.Where(
					fmt.Sprintf("%s BETWEEN ? AND ?", c.Field),
					vals.Index(0).Interface(),
					vals.Index(1).Interface(),
				)
			} else {
				return nil, fmt.Errorf("BETWEEN operator requires a slice of length 2")
			}
		case OpIsNull:
			db = db.Where(fmt.Sprintf("%s IS NULL", c.Field))
		}
	}

	if len(q.builder.selects) > 0 {
		db = db.Select(q.builder.selects)
	}
	if len(q.builder.orderBy) > 0 {
		for _, o := range q.builder.orderBy {
			db = db.Order(fmt.Sprintf("%s %s", o.Field, o.Direction))
		}
	}
	if len(q.builder.groupBy) > 0 {
		for _, g := range q.builder.groupBy {
			db = db.Group(g)
		}
	}
	if q.builder.offset > 0 {
		db = db.Offset(q.builder.offset)
	}
	if q.builder.limit > 0 {
		db = db.Limit(q.builder.limit)
	}

	return db, nil
}

func (q *Query) Builder() *Builder {
	return q.builder
}

func (q *Query) Count(ctx context.Context) (int64, error) {
	db, err := q.compile(ctx)
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
	return db.First(dest).Error
}

func (q *Query) All(ctx context.Context, dest any) error {
	db, err := q.compile(ctx)
	if err != nil {
		return err
	}
	return db.Find(dest).Error
}
