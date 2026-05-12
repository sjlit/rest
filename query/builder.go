package query

type Builder struct {
	conditions []Condition
	selects    []string
	table      string
	joins      []Join
	orderBy    []Order
	groupBy    []string
	limit      int
	offset     int
}

func (q *Builder) Where(field string, op Operator, value any) *Builder {
	q.conditions = append(q.conditions, Condition{Field: field, Op: op, Value: value})
	return q
}

func (q *Builder) AndWhere(conditions ...Condition) *Builder {
	for _, c := range conditions {
		q.conditions = append(q.conditions, c)
	}
	return q
}

func (q *Builder) Select(fields ...string) *Builder {
	q.selects = append(q.selects, fields...)
	return q
}

func (q *Builder) From(table string) *Builder {
	q.table = table
	return q
}

func (q *Builder) OrderBy(field, direction string) *Builder {
	if direction != "DESC" {
		direction = "ASC"
	}
	q.orderBy = append(q.orderBy, Order{Field: field, Direction: direction})
	return q
}

func (q *Builder) GroupBy(fields ...string) *Builder {
	q.groupBy = append(q.groupBy, fields...)
	return q
}

func (q *Builder) Limit(n int) *Builder {
	q.limit = n
	return q
}

func (q *Builder) Offset(n int) *Builder {
	q.offset = n
	return q
}

func NewBuilder() *Builder {
	return &Builder{
		conditions: make([]Condition, 0),
		selects:    make([]string, 0),
		joins:      make([]Join, 0),
		orderBy:    make([]Order, 0),
		groupBy:    make([]string, 0),
	}
}
