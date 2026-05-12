package query

import "testing"

func TestTypesCompile(t *testing.T) {
	_ = QuerySpec{
		Source: TableSource("users"),
		Selects: []Expr{
			Field("name"),
			Count("id").As("total"),
		},
		Where: []Clause{
			Condition{Field: "age", Op: OpGt, Value: 18},
		},
		Having: []Clause{
			Condition{Field: "total", Op: OpGt, Value: 100},
		},
	}
	_ = Subquery{Spec: QuerySpec{Source: TableSource("orders")}}
	_ = SubquerySource{Subquery: Subquery{Alias: "t"}}
}
