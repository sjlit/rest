package query

import "testing"

func TestBuilderSelectExpr(t *testing.T) {
    b := NewBuilder().Select(Field("name"), Count("id").As("total"))
    if len(b.Spec().Selects) != 2 {
        t.Fatalf("expected 2 selects, got %d", len(b.Spec().Selects))
    }
    agg, ok := b.Spec().Selects[1].(AggregateExpr)
    if !ok {
        t.Fatalf("expected AggregateExpr, got %T", b.Spec().Selects[1])
    }
    if agg.Alias != "total" {
        t.Errorf("expected alias 'total', got %s", agg.Alias)
    }
}

func TestBuilderFromSubquery(t *testing.T) {
    b := NewBuilder().FromSubquery("t", func(sb *Builder) {
        sb.Select(Field("user_id")).From("orders")
    })
    src, ok := b.Spec().Source.(SubquerySource)
    if !ok {
        t.Fatalf("expected SubquerySource, got %T", b.Spec().Source)
    }
    if src.Subquery.Alias != "t" {
        t.Errorf("expected alias 't', got %s", src.Subquery.Alias)
    }
}

func TestBuilderHaving(t *testing.T) {
    b := NewBuilder().
        GroupBy("status").
        Having("total", OpGt, 100)
    if len(b.Spec().Having) != 1 {
        t.Fatalf("expected 1 having clause, got %d", len(b.Spec().Having))
    }
}

func TestBuilderWhereExists(t *testing.T) {
    b := NewBuilder().WhereExists(func(sb *Builder) {
        sb.Select(Raw("1")).From("orders")
    })
    if len(b.Spec().Where) != 1 {
        t.Fatalf("expected 1 where clause, got %d", len(b.Spec().Where))
    }
    _, ok := b.Spec().Where[0].(SubqueryClause)
    if !ok {
        t.Fatalf("expected SubqueryClause, got %T", b.Spec().Where[0])
    }
}

func TestBuilderClone(t *testing.T) {
    b := NewBuilder().
        Select(Field("name")).
        Where("age", OpGt, 18).
        Limit(10).
        Offset(5).
        OrderBy("id", "ASC")
    cloned := b.Clone()

    // Modify clone should not affect original
    cloned.Where("name", OpEq, "alice").Limit(20).Offset(10)

    if len(b.Spec().Where) != 1 {
        t.Errorf("original where mutated: expected 1, got %d", len(b.Spec().Where))
    }
    if b.Spec().Limit != 10 {
        t.Errorf("original limit mutated: expected 10, got %d", b.Spec().Limit)
    }
    if b.Spec().Offset != 5 {
        t.Errorf("original offset mutated: expected 5, got %d", b.Spec().Offset)
    }
    if len(b.Spec().Selects) != 1 {
        t.Errorf("original selects mutated: expected 1, got %d", len(b.Spec().Selects))
    }
    if len(b.Spec().OrderBy) != 1 {
        t.Errorf("original orderby mutated: expected 1, got %d", len(b.Spec().OrderBy))
    }
    if cloned.Spec().Limit != 20 {
        t.Errorf("clone limit wrong: expected 20, got %d", cloned.Spec().Limit)
    }
    if cloned.Spec().Offset != 10 {
        t.Errorf("clone offset wrong: expected 10, got %d", cloned.Spec().Offset)
    }
    if len(cloned.Spec().Where) != 2 {
        t.Errorf("clone where wrong: expected 2, got %d", len(cloned.Spec().Where))
    }
}
