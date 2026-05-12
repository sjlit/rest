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
