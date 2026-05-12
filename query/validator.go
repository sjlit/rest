package query

import (
	"fmt"
	"regexp"
	"strings"
)

var safeFieldNameV2 = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

func validateFieldNameV2(field string) error {
	if field == "" {
		return fmt.Errorf("field name cannot be empty")
	}
	if !safeFieldNameV2.MatchString(field) {
		return fmt.Errorf("invalid field name: %s", field)
	}
	return nil
}

func validateTableNameV2(table string) error {
	if table == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if !safeFieldNameV2.MatchString(table) {
		return fmt.Errorf("invalid table name: %s", table)
	}
	return nil
}

var safeJoinOn = regexp.MustCompile(`^[a-zA-Z0-9_\.\s\=\<\>\!\?\,]+$`)

func validateJoinOn(on string) error {
	if on == "" {
		return fmt.Errorf("join condition cannot be empty")
	}
	if strings.Contains(on, ";") || strings.Contains(on, "--") || strings.Contains(on, "/*") {
		return fmt.Errorf("invalid join condition: contains forbidden characters")
	}
	if !safeJoinOn.MatchString(on) {
		return fmt.Errorf("invalid join condition")
	}
	return nil
}

var allowedAggregates = map[string]bool{
	"COUNT": true, "SUM": true, "AVG": true, "MAX": true, "MIN": true,
}

func Validate(spec QuerySpec) error {
	// Source
	if spec.Source != nil {
		switch s := spec.Source.(type) {
		case TableSource:
			if err := validateTableNameV2(string(s)); err != nil {
				return err
			}
		case SubquerySource:
			if err := validateTableNameV2(s.Subquery.Alias); err != nil {
				return err
			}
			if err := Validate(s.Subquery.Spec); err != nil {
				return err
			}
		}
	}

	// Selects
	for _, e := range spec.Selects {
		if err := validateExpr(e); err != nil {
			return err
		}
	}

	// Where
	for _, c := range spec.Where {
		if err := validateClause(c); err != nil {
			return err
		}
	}

	// Joins
	for _, j := range spec.Joins {
		if err := validateTableNameV2(j.Table); err != nil {
			return err
		}
		if err := validateJoinOn(j.On); err != nil {
			return err
		}
	}

	// OrderBy
	for _, o := range spec.OrderBy {
		if err := validateFieldNameV2(o.Field); err != nil {
			return err
		}
	}

	// GroupBy
	for _, g := range spec.GroupBy {
		if err := validateFieldNameV2(g); err != nil {
			return err
		}
	}

	// Having
	for _, c := range spec.Having {
		if err := validateClause(c); err != nil {
			return err
		}
	}

	return nil
}

func validateExpr(e Expr) error {
	switch ex := e.(type) {
	case FieldExpr:
		return validateFieldNameV2(ex.Name)
	case AggregateExpr:
		if err := validateFieldNameV2(ex.Field); err != nil {
			return err
		}
		if !allowedAggregates[ex.Function] {
			return fmt.Errorf("invalid aggregate function: %s", ex.Function)
		}
		return nil
	case AliasedExpr:
		return validateExpr(ex.Expr)
	case RawExpr:
		if strings.Contains(ex.SQL, ";") || strings.Contains(ex.SQL, "--") || strings.Contains(ex.SQL, "/*") {
			return fmt.Errorf("invalid raw SQL: contains forbidden characters")
		}
		return nil
	default:
		return fmt.Errorf("unknown expr type: %T", e)
	}
}

func validateClause(clause Clause) error {
	switch c := clause.(type) {
	case Condition:
		if err := validateFieldNameV2(c.Field); err != nil {
			return err
		}
		if sub, ok := c.Value.(Subquery); ok {
			return Validate(sub.Spec)
		}
		return nil
	case AndGroup:
		for _, cl := range c.Clauses {
			if err := validateClause(cl); err != nil {
				return err
			}
		}
	case OrGroup:
		for _, cl := range c.Clauses {
			if err := validateClause(cl); err != nil {
				return err
			}
		}
	case SubqueryClause:
		return Validate(c.Subquery.Spec)
	default:
		return fmt.Errorf("unknown clause type: %T", clause)
	}
	return nil
}
