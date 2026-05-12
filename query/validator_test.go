package query

import (
	"errors"
	"testing"
)

func TestValidateFieldName(t *testing.T) {
	tests := []struct {
		field string
		want  error
	}{
		{"name", nil},
		{"users.name", nil},
		{"name_1", nil},
		{"", errors.New("")},
		{"name; DROP TABLE users", errors.New("")},
		{"name--", errors.New("")},
		{"name' OR '1'='1", errors.New("")},
	}

	for _, tt := range tests {
		err := validateFieldName(tt.field)
		if tt.want == nil && err != nil {
			t.Errorf("validateFieldName(%q) unexpected error: %v", tt.field, err)
		}
		if tt.want != nil && err == nil {
			t.Errorf("validateFieldName(%q) expected error, got nil", tt.field)
		}
	}
}

func TestValidateBlocksInjection(t *testing.T) {
	cases := []struct {
		name string
		spec QuerySpec
	}{
		{"where_field", QuerySpec{Where: []Clause{Condition{Field: "id; DROP", Op: OpEq, Value: 1}}}},
		{"order_by", QuerySpec{OrderBy: []Order{{Field: "name;--", Direction: "ASC"}}}},
		{"group_by", QuerySpec{GroupBy: []string{"name;--"}}},
		{"select", QuerySpec{Selects: []Expr{Field("id;--")}}},
		{"join_table", QuerySpec{Joins: []Join{{Table: "users;--", On: "a = b"}}}},
		{"join_on", QuerySpec{Joins: []Join{{Table: "orders", On: "1=1; DROP TABLE users--"}}}},
		{"nested_where", QuerySpec{Where: []Clause{AndGroup{Clauses: []Clause{Condition{Field: "id;--", Op: OpEq, Value: 1}}}}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.spec)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestValidateAggregateWhitelist(t *testing.T) {
	spec := QuerySpec{
		Selects: []Expr{AggregateExpr{Field: "id", Function: "INVALID"}},
	}
	if err := Validate(spec); err == nil {
		t.Error("expected error for invalid aggregate function")
	}
}

func TestValidateSubqueryRecursive(t *testing.T) {
	spec := QuerySpec{
		Where: []Clause{Condition{
			Field: "id",
			Op:    OpIn,
			Value: Subquery{Spec: QuerySpec{
				Source: TableSource("orders;--"),
			}},
		}},
	}
	if err := Validate(spec); err == nil {
		t.Error("expected validation error for subquery with bad table name")
	}
}
