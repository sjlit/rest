package query

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type compilerFunc func(db *gorm.DB, field string, value any) (*gorm.DB, error)

type Compiler struct {
	operators map[Operator]compilerFunc
}

func NewCompiler() *Compiler {
	c := &Compiler{operators: make(map[Operator]compilerFunc)}
	c.operators[OpEq] = c.compileEq
	c.operators[OpNe] = c.compileNe
	c.operators[OpGt] = c.compileSimple(">")
	c.operators[OpLt] = c.compileSimple("<")
	c.operators[OpGte] = c.compileSimple(">=")
	c.operators[OpLte] = c.compileSimple("<=")
	c.operators[OpLike] = c.compileLike
	c.operators[OpIn] = c.compileIn
	c.operators[OpBetween] = c.compileBetween
	c.operators[OpIsNull] = c.compileIsNull
	c.operators[OpIsNotNull] = c.compileIsNotNull
	return c
}

const OpNotIn Operator = "NOT IN"

func (c *Compiler) Compile(spec QuerySpec, db *gorm.DB) (*gorm.DB, error) {
	// Source
	if spec.Source != nil {
		switch s := spec.Source.(type) {
		case TableSource:
			db = db.Table(string(s))
		case SubquerySource:
			subSQL, subArgs, err := c.compileSubquery(s.Subquery, db)
			if err != nil {
				return nil, err
			}
			db = db.Table(fmt.Sprintf("(%s) AS %s", subSQL, s.Subquery.Alias), subArgs...)
		}
	}

	// Joins
	for _, j := range spec.Joins {
		joinSQL := fmt.Sprintf("%s JOIN %s ON %s", j.Direction, j.Table, j.On)
		db = db.Joins(joinSQL, j.Args...)
	}

	// Where
	db, err := c.compileClauses(db, spec.Where)
	if err != nil {
		return nil, err
	}

	// Select — combine all expressions into a single Select call so that
	// RawExpr args (e.g. Raw("COALESCE(?, id)", 100)) can be bound as
	// variadic placeholders, instead of being silently dropped.
	if len(spec.Selects) > 0 {
		parts := make([]string, 0, len(spec.Selects))
		var allArgs []any
		for _, e := range spec.Selects {
			sql, args := c.compileExpr(e)
			parts = append(parts, sql)
			allArgs = append(allArgs, args...)
		}
		db = db.Select(strings.Join(parts, ", "), allArgs...)
	}

	// GroupBy
	for _, g := range spec.GroupBy {
		db = db.Group(g)
	}

	// Having
	db, err = c.compileHavingClauses(db, spec.Having)
	if err != nil {
		return nil, err
	}

	// OrderBy
	for _, o := range spec.OrderBy {
		db = db.Order(fmt.Sprintf("%s %s", o.Field, o.Direction))
	}

	// Limit / Offset
	if spec.Offset > 0 {
		db = db.Offset(spec.Offset)
	}
	if spec.Limit > 0 {
		db = db.Limit(spec.Limit)
	}

	return db, nil
}

func (c *Compiler) compileClauses(db *gorm.DB, clauses []Clause) (*gorm.DB, error) {
	if len(clauses) == 0 {
		return db, nil
	}

	var err error
	db, err = c.compileClauseAsAnd(db, clauses[0])
	if err != nil {
		return nil, err
	}

	for _, clause := range clauses[1:] {
		switch cl := clause.(type) {
		case Condition, AndGroup:
			db, err = c.compileClauseAsAnd(db, cl)
		case OrGroup:
			db, err = c.compileClauseAsOr(db, cl)
		case SubqueryClause:
			db, err = c.compileSubqueryClause(db, cl)
		}
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (c *Compiler) compileClauseAsAnd(db *gorm.DB, clause Clause) (*gorm.DB, error) {
	switch cl := clause.(type) {
	case Condition:
		return c.compileCondition(db, cl)
	case AndGroup:
		sub, err := c.compileClauses(db.Session(&gorm.Session{NewDB: true}), cl.Clauses)
		if err != nil {
			return nil, err
		}
		return db.Where(sub), nil
	case SubqueryClause:
		return c.compileSubqueryClause(db, cl)
	}
	return nil, fmt.Errorf("unexpected clause type in AND context: %T", clause)
}

func (c *Compiler) compileClauseAsOr(db *gorm.DB, clause OrGroup) (*gorm.DB, error) {
	sub, err := c.compileClauses(db.Session(&gorm.Session{NewDB: true}), clause.Clauses)
	if err != nil {
		return nil, err
	}
	return db.Or(sub), nil
}

func (c *Compiler) compileHavingClauses(db *gorm.DB, clauses []Clause) (*gorm.DB, error) {
	if len(clauses) == 0 {
		return db, nil
	}

	var err error
	db, err = c.compileHavingClauseAsAnd(db, clauses[0])
	if err != nil {
		return nil, err
	}

	for _, clause := range clauses[1:] {
		switch cl := clause.(type) {
		case Condition, AndGroup:
			db, err = c.compileHavingClauseAsAnd(db, cl)
		case OrGroup:
			db, err = c.compileHavingClauseAsOr(db, cl)
		case SubqueryClause:
			db, err = c.compileSubqueryClause(db, cl)
		}
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (c *Compiler) compileHavingClauseAsAnd(db *gorm.DB, clause Clause) (*gorm.DB, error) {
	switch cl := clause.(type) {
	case Condition:
		return c.compileHavingCondition(db, cl)
	case AndGroup:
		sub, err := c.compileHavingClauses(db.Session(&gorm.Session{NewDB: true}), cl.Clauses)
		if err != nil {
			return nil, err
		}
		return db.Having(sub), nil
	case SubqueryClause:
		return c.compileSubqueryClause(db, cl)
	}
	return nil, fmt.Errorf("unexpected clause type in HAVING AND context: %T", clause)
}

func (c *Compiler) compileHavingClauseAsOr(db *gorm.DB, clause OrGroup) (*gorm.DB, error) {
	sub, err := c.compileHavingClauses(db.Session(&gorm.Session{NewDB: true}), clause.Clauses)
	if err != nil {
		return nil, err
	}
	return db.Or(sub), nil
}

func (c *Compiler) compileHavingCondition(db *gorm.DB, cond Condition) (*gorm.DB, error) {
	// Handle subquery IN/NOT IN at the condition level
	if sub, ok := cond.Value.(Subquery); ok {
		subSQL, subArgs, err := c.compileSubquery(sub, db)
		if err != nil {
			return nil, err
		}
		switch cond.Op {
		case OpIn:
			return db.Having(fmt.Sprintf("%s IN (%s)", cond.Field, subSQL), subArgs...), nil
		case OpNotIn:
			return db.Having(fmt.Sprintf("%s NOT IN (%s)", cond.Field, subSQL), subArgs...), nil
		default:
			return nil, fmt.Errorf("subquery only supported with IN/NOT IN, got: %s", cond.Op)
		}
	}

	return c.compileHavingSimple(db, cond.Field, cond.Op, cond.Value)
}

func (c *Compiler) compileHavingSimple(db *gorm.DB, field string, op Operator, value any) (*gorm.DB, error) {
	switch op {
	case OpEq:
		if value == nil {
			return db.Having(fmt.Sprintf("%s IS NULL", field)), nil
		}
		return db.Having(fmt.Sprintf("%s = ?", field), value), nil
	case OpNe:
		if value == nil {
			return db.Having(fmt.Sprintf("%s IS NOT NULL", field)), nil
		}
		return db.Having(fmt.Sprintf("%s <> ?", field), value), nil
	case OpGt:
		return db.Having(fmt.Sprintf("%s > ?", field), value), nil
	case OpLt:
		return db.Having(fmt.Sprintf("%s < ?", field), value), nil
	case OpGte:
		return db.Having(fmt.Sprintf("%s >= ?", field), value), nil
	case OpLte:
		return db.Having(fmt.Sprintf("%s <= ?", field), value), nil
	case OpLike:
		return db.Having(fmt.Sprintf("%s LIKE ? ESCAPE '\\'", field), value), nil
	case OpIn:
		return db.Having(fmt.Sprintf("%s IN ?", field), value), nil
	case OpBetween:
		v := reflect.ValueOf(value)
		if v.Kind() != reflect.Slice || v.Len() != 2 {
			return nil, fmt.Errorf("BETWEEN operator requires a slice of length 2")
		}
		return db.Having(fmt.Sprintf("%s BETWEEN ? AND ?", field), v.Index(0).Interface(), v.Index(1).Interface()), nil
	case OpIsNull:
		return db.Having(fmt.Sprintf("%s IS NULL", field)), nil
	case OpIsNotNull:
		return db.Having(fmt.Sprintf("%s IS NOT NULL", field)), nil
	default:
		return nil, fmt.Errorf("unsupported operator for HAVING: %s", op)
	}
}

func (c *Compiler) compileSubqueryClause(db *gorm.DB, clause SubqueryClause) (*gorm.DB, error) {
	subSQL, subArgs, err := c.compileSubquery(clause.Subquery, db)
	if err != nil {
		return nil, err
	}
	switch clause.Op {
	case OpExists:
		return db.Where(fmt.Sprintf("EXISTS (%s)", subSQL), subArgs...), nil
	case OpNotExists:
		return db.Where(fmt.Sprintf("NOT EXISTS (%s)", subSQL), subArgs...), nil
	default:
		return nil, fmt.Errorf("unsupported subquery operator: %s", clause.Op)
	}
}

func (c *Compiler) compileCondition(db *gorm.DB, cond Condition) (*gorm.DB, error) {
	// Handle subquery IN/NOT IN at the condition level
	if sub, ok := cond.Value.(Subquery); ok {
		subSQL, subArgs, err := c.compileSubquery(sub, db)
		if err != nil {
			return nil, err
		}
		switch cond.Op {
		case OpIn:
			return db.Where(fmt.Sprintf("%s IN (%s)", cond.Field, subSQL), subArgs...), nil
		case OpNotIn:
			return db.Where(fmt.Sprintf("%s NOT IN (%s)", cond.Field, subSQL), subArgs...), nil
		default:
			return nil, fmt.Errorf("subquery only supported with IN/NOT IN, got: %s", cond.Op)
		}
	}

	compiler, ok := c.operators[cond.Op]
	if !ok {
		return nil, fmt.Errorf("unsupported operator: %s", cond.Op)
	}
	return compiler(db, cond.Field, cond.Value)
}

func (c *Compiler) compileSubquery(sub Subquery, parentDB *gorm.DB) (string, []any, error) {
	dryDB := parentDB.Session(&gorm.Session{NewDB: true, DryRun: true})
	compiled, err := c.Compile(sub.Spec, dryDB)
	if err != nil {
		return "", nil, err
	}

	var dummy []map[string]any
	stmt := compiled.Find(&dummy).Statement

	return stmt.SQL.String(), stmt.Vars, nil
}

func (c *Compiler) compileExpr(e Expr) (string, []any) {
	switch ex := e.(type) {
	case FieldExpr:
		return ex.Name, nil
	case AggregateExpr:
		sql := fmt.Sprintf("%s(%s)", ex.Function, ex.Field)
		if ex.Alias != "" {
			sql = fmt.Sprintf("%s AS %s", sql, ex.Alias)
		}
		return sql, nil
	case AliasedExpr:
		inner, args := c.compileExpr(ex.Expr)
		return fmt.Sprintf("%s AS %s", inner, ex.Alias), args
	case RawExpr:
		return ex.SQL, ex.Args
	default:
		return "", nil
	}
}

// ---- Operator Compilers (methods on Compiler to avoid name conflicts) ----

func (c *Compiler) compileEq(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	if value == nil {
		return db.Where(fmt.Sprintf("%s IS NULL", field)), nil
	}
	return db.Where(fmt.Sprintf("%s = ?", field), value), nil
}

func (c *Compiler) compileNe(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	if value == nil {
		return db.Where(fmt.Sprintf("%s IS NOT NULL", field)), nil
	}
	return db.Where(fmt.Sprintf("%s <> ?", field), value), nil
}

func (c *Compiler) compileSimple(op string) compilerFunc {
	return func(db *gorm.DB, field string, value any) (*gorm.DB, error) {
		return db.Where(fmt.Sprintf("%s %s ?", field, op), value), nil
	}
}

func (c *Compiler) compileLike(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s LIKE ? ESCAPE '\\'", field), value), nil
}

func (c *Compiler) compileIn(db *gorm.DB, field string, value any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IN ?", field), value), nil
}

func (c *Compiler) compileBetween(db *gorm.DB, field string, value any) (*gorm.DB, error) {
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

func (c *Compiler) compileIsNull(db *gorm.DB, field string, _ any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IS NULL", field)), nil
}

func (c *Compiler) compileIsNotNull(db *gorm.DB, field string, _ any) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%s IS NOT NULL", field)), nil
}
