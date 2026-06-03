package query

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	return db
}

func TestCompileBasicWhere(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source: TableSource("users"),
		Where:  []Clause{Condition{Field: "age", Op: OpGt, Value: 18}},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "age > 18") {
		t.Errorf("expected SQL to contain 'age > 18', got: %s", sql)
	}
}

func TestCompileSelectAggregate(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source:  TableSource("orders"),
		Selects: []Expr{Field("status"), Count("id").As("total")},
		GroupBy: []string{"status"},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "COUNT(id) AS total") {
		t.Errorf("expected aggregate SQL, got: %s", sql)
	}
}

func TestCompileFromSubquery(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source: SubquerySource{Subquery: Subquery{
			Alias: "t",
			Spec: QuerySpec{
				Source:  TableSource("orders"),
				Selects: []Expr{Field("user_id")},
			},
		}},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "SELECT user_id FROM `orders`") {
		t.Errorf("expected subquery SQL, got: %s", sql)
	}
}

func TestCompileWhereInSubquery(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source: TableSource("users"),
		Where: []Clause{Condition{
			Field: "id",
			Op:    OpIn,
			Value: Subquery{Spec: QuerySpec{
				Source:  TableSource("orders"),
				Selects: []Expr{Field("user_id")},
				Where:   []Clause{Condition{Field: "amount", Op: OpGt, Value: 100}},
			}},
		}},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "id IN (SELECT user_id FROM `orders` WHERE amount > 100)") {
		t.Errorf("expected IN subquery SQL, got: %s", sql)
	}
}

func TestCompileWhereExists(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source: TableSource("users"),
		Where: []Clause{SubqueryClause{
			Op: OpExists,
			Subquery: Subquery{Spec: QuerySpec{
				Source:  TableSource("orders"),
				Selects: []Expr{Raw("1")},
			}},
		}},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "EXISTS (SELECT 1 FROM `orders`)") && !strings.Contains(sql, "EXISTS (SELECT 1 FROM \"orders\")") {
		t.Errorf("expected EXISTS SQL, got: %s", sql)
	}
}

func TestCompileHaving(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source:  TableSource("orders"),
		Selects: []Expr{Field("status"), Sum("amount").As("total")},
		GroupBy: []string{"status"},
		Having: []Clause{Condition{
			Field: "total",
			Op:    OpGt,
			Value: 1000,
		}},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	sql := compiled.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]map[string]any{})
	})
	if !strings.Contains(sql, "HAVING total > 1000") {
		t.Errorf("expected HAVING SQL, got: %s", sql)
	}
}

func TestCompileSelectRawExprArgs(t *testing.T) {
	db := setupDryRunDB(t)
	spec := QuerySpec{
		Source:  TableSource("orders"),
		Selects: []Expr{Raw("COALESCE(?, id)", 100)},
	}
	c := NewCompiler()
	compiled, err := c.Compile(spec, db.Session(&gorm.Session{DryRun: true}))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	// Build the SQL by calling Find on a dryrun session, then inspect the
	// raw Statement. ToSQL would inline the bound Vars, so we check the
	// underlying SQL/Vars directly.
	compiled.Find(&[]map[string]any{})

	// The placeholder ? must be preserved in the raw SQL (not expanded to
	// the literal 100), and the corresponding value 100 must appear in Vars.
	rawSQL := compiled.Statement.SQL.String()
	if !strings.Contains(rawSQL, "COALESCE(?, id)") {
		t.Errorf("expected raw SQL with placeholder preserved, got: %s", rawSQL)
	}
	bindings := compiled.Statement.Vars
	found := false
	for _, v := range bindings {
		if v == int64(100) || v == int(100) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected arg 100 in bindings, got: %v", bindings)
	}
}
