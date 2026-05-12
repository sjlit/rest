# Query Package Redesign — Design Document

**Date:** 2026-05-12
**Scope:** `query/` package
**Goals:**
1. Add subquery and aggregate/HAVING capabilities.
2. Split Builder / AST / Compiler / Validator into clean, independently testable layers.
3. Keep GORM coupling (no abstraction over DB drivers).

---

## 1. Architecture

The package is split into 5 files with single, well-defined responsibilities:

| File | Responsibility | Dependencies |
|------|---------------|--------------|
| `ast.go` | Pure data structures: `QuerySpec`, `Clause`, `Expr`, `Join`, `Order`, `Subquery` | Standard library only |
| `builder.go` | Fluent API. Builds `QuerySpec`. Zero validation / compilation logic. | `ast.go` |
| `validator.go` | Validates a `QuerySpec` (injection prevention, format checks). | `ast.go` |
| `compiler.go` | Compiles `QuerySpec` → `*gorm.DB`. Operator registry. Subquery SQL generation. | `ast.go`, `gorm` |
| `query.go` | Executor: `Count` / `One` / `All` / `Page`. Orchestrates validator → compiler → gorm. | All above |

---

## 2. AST (`ast.go`)

### 2.1 QuerySpec — The Root AST Node

Replaces the scattered fields inside `Builder`:

```go
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
```

`Source` is an interface representing the query's data source:

```go
type Source interface {
    isSource()
}

type TableSource string
func (TableSource) isSource() {}

type SubquerySource struct {
    Subquery Subquery
}
func (SubquerySource) isSource() {}
```

### 2.2 Expr — Selectable Expressions

All projections (SELECT list, ORDER BY) are expressed through the `Expr` interface:

```go
type Expr interface {
    isExpr()
}

type FieldExpr struct {
    Name string
}

// AliasedExpr wraps any Expr with an alias
type AliasedExpr struct {
    Expr  Expr
    Alias string
}

// AggregateExpr implements Expr and provides As() for chaining
type AggregateExpr struct {
    Field    string
    Function string // "COUNT", "SUM", "AVG", "MAX", "MIN"
    Alias    string // optional; set via As()
}

func (a AggregateExpr) As(alias string) Expr {
    a.Alias = alias
    return a
}

// RawExpr for complex SQL fragments
type RawExpr struct {
    SQL  string
    Args []any
}
```

Helper functions. `Count`, `Sum`, `Avg`, `Max`, `Min` return `AggregateExpr` so `.As()` can be chained:

```go
func Field(name string) Expr
func Count(field string) AggregateExpr
func Sum(field string) AggregateExpr
func Avg(field string) AggregateExpr
func Max(field string) AggregateExpr
func Min(field string) AggregateExpr
func Raw(sql string, args ...any) Expr
```

Usage:

```go
b.Select(Field("name"), Count("id").As("total"))
```

### 2.3 Clause — WHERE / HAVING Conditions

Retain existing types; add subquery support:

```go
type Clause interface {
    isClause()
}

type Condition struct {
    Field string
    Op    Operator
    Value any
}

// AndGroup and OrGroup remain unchanged
type AndGroup struct { Clauses []Clause }
type OrGroup struct { Clauses []Clause }

// Subquery as a condition (for EXISTS / NOT EXISTS)
type SubqueryClause struct {
    Op       Operator // OpExists, OpNotExists
    Subquery Subquery
}
```

`Subquery` is also used directly as a `Condition.Value` for `OpIn` / `OpNotIn`:

```go
func Subquery(fn func(*Builder)) Subquery
```

### 2.4 Supporting Types

```go
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

type Subquery struct {
    Spec  QuerySpec
    Alias string
}
```

---

## 3. Builder API (`builder.go`)

Builder holds a `QuerySpec` and exposes a fluent API:

```go
type Builder struct {
    spec QuerySpec
}

func NewBuilder() *Builder
func (b *Builder) Spec() QuerySpec
```

### 3.1 Projection

```go
func (b *Builder) Select(exprs ...Expr) *Builder
```

Usage:

```go
b.Select(Field("name"), Count("id").As("total"))
```

### 3.2 Source

```go
func (b *Builder) From(table string) *Builder
func (b *Builder) FromSubquery(alias string, fn func(*Builder)) *Builder
```

Usage:

```go
b.From("users")

b.FromSubquery("t", func(sb *Builder) {
    sb.Select(Field("user_id"), Count("*").As("cnt")).
      From("orders").
      GroupBy("user_id")
})
```

### 3.3 Joins

Unchanged from current API:

```go
func (b *Builder) Join(direction, table, on string, args ...any) *Builder
func (b *Builder) LeftJoin(table, on string, args ...any) *Builder
func (b *Builder) InnerJoin(table, on string, args ...any) *Builder
```

### 3.4 Where Conditions

Unchanged signatures; `OpIn` / `OpNotIn` now accept `Subquery` as `Value`:

```go
func (b *Builder) Where(field string, op Operator, value any) *Builder
func (b *Builder) OrWhere(field string, op Operator, value any) *Builder
func (b *Builder) WhereGroup(fn func(*Builder)) *Builder
func (b *Builder) OrWhereGroup(fn func(*Builder)) *Builder
func (b *Builder) Like(field string, value string, mode LikeMode) *Builder

// Subquery-specific helpers (EXISTS has no field parameter)
func (b *Builder) WhereExists(fn func(*Builder)) *Builder
func (b *Builder) WhereNotExists(fn func(*Builder)) *Builder
```

Usage:

```go
// IN subquery
b.Where("user_id", OpIn, Subquery(func(sb *Builder) {
    sb.Select(Field("user_id")).From("orders").Where("amount", OpGt, 100)
}))

// EXISTS
b.WhereExists(Subquery(func(sb *Builder) {
    sb.Select(Raw("1")).From("orders").
      Where("orders.user_id", OpEq, Raw("users.id"))
}))
```

### 3.5 Ordering, Grouping, Pagination

```go
func (b *Builder) OrderBy(field, direction string) *Builder
func (b *Builder) GroupBy(fields ...string) *Builder
func (b *Builder) Limit(n int) *Builder
func (b *Builder) Offset(n int) *Builder
```

### 3.6 Having

Mirrors the `Where` API exactly. Conditions are appended to `spec.Having`:

```go
func (b *Builder) Having(field string, op Operator, value any) *Builder
func (b *Builder) OrHaving(field string, op Operator, value any) *Builder
func (b *Builder) HavingGroup(fn func(*Builder)) *Builder
func (b *Builder) OrHavingGroup(fn func(*Builder)) *Builder
```

Usage:

```go
b.GroupBy("status").
  Select(Field("status"), Sum("amount").As("total")).
  Having("total", OpGt, 1000)
```

---

## 4. Validator (`validator.go`)

A pure function with no external dependencies:

```go
func Validate(spec QuerySpec) error
```

Validation rules:
- `Source`: table names / aliases match `^[a-zA-Z0-9_]+$`
- `Selects`: each `Expr` validated recursively
  - `FieldExpr.Name`: `^[a-zA-Z0-9_\.]+$`
  - `AggregateExpr.Field`: same as above; `Function` must be in whitelist `{COUNT, SUM, AVG, MAX, MIN}`
  - `RawExpr.SQL`: basic forbidden-pattern check (`;`, `--`, `/*`)
  - `AliasedExpr`: validate inner `Expr`
- `Where` / `Having`: recursively validate each `Clause`
  - `Condition.Field`: `^[a-zA-Z0-9_\.]+$`
  - `Condition.Value`: if value is `Subquery`, recursively `Validate(subquery.Spec)`
  - `SubqueryClause.Subquery`: recursively validate `Spec`
- `Joins`: `Table` name and `On` condition validated (same rules as current)
- `OrderBy` / `GroupBy`: field names validated

`Where` and `Having` reuse the **same** `validateClause` recursion.

---

## 5. Compiler (`compiler.go`)

### 5.1 Structure

```go
type Compiler struct {
    operators map[Operator]conditionCompiler
}

func NewCompiler() *Compiler
func (c *Compiler) Compile(spec QuerySpec, db *gorm.DB) (*gorm.DB, error)
```

`conditionCompiler` signature remains:

```go
type conditionCompiler func(db *gorm.DB, field string, value any) (*gorm.DB, error)
```

### 5.2 Compilation Order

Matches SQL syntax evaluation order:

1. `Source` — `db.Table(...)` or subquery SQL
2. `Joins` — `db.Joins(...)`
3. `Where` — `compileClauses(db, spec.Where)`
4. `Select` — `db.Select(...)` with Expr SQL generation
5. `GroupBy` — `db.Group(...)`
6. `Having` — `compileClauses(db, spec.Having)` (reuses Where logic)
7. `OrderBy` — `db.Order(...)`
8. `Limit` / `Offset`

### 5.3 Subquery Compilation

Subqueries must produce a SQL string + arguments for embedding into outer SQL. Strategy: compile the subquery AST onto a GORM `DryRun` session and extract the generated SQL:

```go
func (c *Compiler) compileSubquery(sub Subquery, parentDB *gorm.DB) (string, []any, error) {
    dryDB := parentDB.Session(&gorm.Session{NewDB: true, DryRun: true})
    compiled, err := c.compileAST(sub.Spec, dryDB)
    if err != nil {
        return "", nil, err
    }

    var dummy []map[string]any
    stmt := compiled.Find(&dummy).Statement

    return stmt.SQL.String(), stmt.Vars, nil
}
```

The outer compiler then embeds:

```go
// For IN subquery:
db.Where(fmt.Sprintf("%s IN (%s)", field, subSQL), subArgs...)

// For EXISTS:
db.Where(fmt.Sprintf("EXISTS (%s)", subSQL), subArgs...)
```

### 5.4 Expr Compilation

Each `Expr` type implements SQL generation:

```go
func compileExpr(e Expr) (string, []any)
```

- `FieldExpr` → `"field_name"`, nil
- `AggregateExpr` → `"FUNCTION(field_name)"`, nil
- `AliasedExpr` → `"expr_sql AS alias"`, nil
- `RawExpr` → `"sql"`, args

### 5.5 Operator Registry

Default operators registered in `NewCompiler()`:

```go
OpEq, OpNe, OpGt, OpLt, OpGte, OpLte, OpLike, OpIn, OpBetween,
OpIsNull, OpIsNotNull
```

`OpIn` / `OpNotIn` extended to handle `Subquery` via type switch.

---

## 6. Query Executor (`query.go`)

Stripped down to orchestration only:

```go
type Query struct {
    db      *gorm.DB
    model   any
    builder *Builder
}

func New(db *gorm.DB, model any, builder *Builder) *Query
func (q *Query) Builder() *Builder

func (q *Query) Count(ctx context.Context) (int64, error)
func (q *Query) One(ctx context.Context, dest any) error
func (q *Query) All(ctx context.Context, dest any) error
func (q *Query) Page(ctx context.Context, page, size int, dest any) (int64, error)
```

The `compile` private method becomes:

```go
func (q *Query) compile(ctx context.Context) (*gorm.DB, error) {
    if err := validator.Validate(q.builder.spec); err != nil {
        return nil, err
    }

    db := q.db.WithContext(ctx)
    compiler := NewCompiler()
    return compiler.Compile(q.builder.spec, db)
}
```

---

## 7. Testing Strategy

Four independent test layers:

| Layer | File | What | Needs DB? |
|-------|------|------|-----------|
| AST | `builder_test.go` | Assert `Builder` produces correct `QuerySpec` | No |
| Validator | `validator_test.go` | Table-driven: invalid spec → error | No |
| Compiler | `compiler_test.go` | DryRun session: assert generated SQL strings | No (DryRun) |
| Integration | `query_test.go` | End-to-end with SQLite: execute and verify results | Yes (SQLite in-memory) |

### 7.1 Migration of Existing Tests

- `TestValidateFieldName` → `validator_test.go`
- `TestBuilderValidationBlocksInjection` → `validator_test.go`
- Operator tests (`TestOpEq`, `TestOpGt`, etc.) → `compiler_test.go` (assert SQL)
- Execution tests (`TestCount`, `TestPage`, etc.) → remain in `query_test.go`

### 7.2 New Test Coverage

- Subquery in `FROM`: `TestCompileFromSubquery`
- Subquery in `WHERE IN`: `TestCompileWhereInSubquery`
- `EXISTS`: `TestCompileWhereExists`
- Aggregate in `SELECT`: `TestCompileSelectAggregate`
- `HAVING`: `TestCompileHaving`
- Validation: `TestValidateSubquery`, `TestValidateAggregateFunctionWhitelist`

---

## 8. Migration Plan

1. **Create `ast.go`** — Move `Clause`, `Condition`, `AndGroup`, `OrGroup`, `Join`, `Order`, and all new types (`Expr`, `QuerySpec`, `Subquery`, etc.).
2. **Refactor `builder.go`** — Replace internal slices with `QuerySpec`. Update all methods to operate on `b.spec`.
3. **Extract `validator.go`** — Move validation logic from `builder.go` and `query.go`. Create `Validate(spec QuerySpec)`.
4. **Extract `compiler.go`** — Move `compile`, `compileClauses`, `compileCondition`, and operator map from `query.go`. Add subquery / aggregate / having compilation.
5. **Simplify `query.go`** — Remove validation and compilation logic; keep only `Query` struct and execution methods.
6. **Split tests** — Create `builder_test.go`, `validator_test.go`, `compiler_test.go`. Migrate existing tests. Add new coverage.

All changes are internal to the `query` package. Public API signatures remain compatible **except** `Select` which changes from `...string` to `...Expr`. Since compatibility is explicitly not a constraint, this is acceptable.
