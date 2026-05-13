package rest

import (
	"context"
	"encoding/base64"
	"reflect"
	"slices"
	"strconv"

	"git.nobla.cn/golang/rest/internal/inflector"
	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
	gormSchema "gorm.io/gorm/schema"
)

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(b))
}

type Model[T any] struct {
	db         *gorm.DB
	opts       *options
	naming     Naming
	primaryKey string
}

func (m *Model[T]) GetDB() *gorm.DB {
	return m.db
}

func (m *Model[T]) HasScenario(s string) bool {
	if len(m.opts.scenarios) == 0 {
		return true
	}
	return slices.Contains(m.opts.scenarios, s)
}

func (m *Model[T]) GetPrimaryKey() string {
	return m.primaryKey
}

func (m *Model[T]) GetNaming() Naming {
	return m.naming
}

func (m *Model[T]) GetFields() []*gormSchema.Field {
	return m.GetDB().Statement.Schema.Fields
}

func (m *Model[T]) GetFieldValue(refValue reflect.Value, column string) any {
	var (
		rawField *gormSchema.Field
	)
	refVal := reflect.Indirect(refValue)
	for _, field := range m.GetDB().Statement.Schema.Fields {
		if field.DBName == column || field.Name == column {
			rawField = field
			break
		}
	}
	if rawField == nil {
		return nil
	}
	var targetValue reflect.Value
	targetValue = refVal
	for _, i := range rawField.StructField.Index {
		targetValue = targetValue.Field(i)
	}
	return targetValue.Interface()
}

func (m *Model[T]) Create(ctx context.Context, model *T) (diffAttrs []*DiffAttr, err error) {
	if !m.HasScenario(schema.ScenarioCreate) {
		return nil, ErrPermissionDenied
	}
	var (
		schemas []schema.Schema
	)
	if schemas, err = schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioCreate); err != nil {
		return
	}
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope != nil {
		runtimeScope.Schemas = schemas
	}
	if err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		if errTx = tx.Create(model).Error; errTx != nil {
			return
		}
		diffAttrs = make([]*DiffAttr, 0, len(schemas))
		for _, row := range schemas {
			diffAttrs = append(diffAttrs, &DiffAttr{
				Column:   row.Column,
				Label:    row.Label,
				Previous: nil,
				Current:  m.GetFieldValue(reflect.ValueOf(model), row.Column),
			})
		}

		return
	}); err != nil {
		return nil, err
	}
	if ac, ok := any(model).(AfterCreated); ok {
		ac.AfterCreated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := any(model).(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model[T]) Update(ctx context.Context, primaryKey any, model T) (diffAttrs []*DiffAttr, err error) {
	if !m.HasScenario(schema.ScenarioUpdate) {
		return nil, ErrPermissionDenied
	}
	var (
		updates        map[string]any
		previousValues map[string]any
		schemas        []schema.Schema
	)
	if schemas, err = schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioUpdate); err != nil {
		return
	}
	modelValue := reflect.ValueOf(model)
	updates = make(map[string]any)
	previousValues = make(map[string]any)
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope != nil {
		runtimeScope.Schemas = schemas
	}
	err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		previousModel := reflect.New(reflect.Indirect(reflect.ValueOf(model)).Type()).Interface()
		if errTx = tx.Where(map[string]any{m.primaryKey: primaryKey}).First(previousModel).Error; errTx != nil {
			return errTx
		}
		previousModelValue := reflect.ValueOf(previousModel)
		for _, row := range schemas {
			previousValues[row.Column] = m.GetFieldValue(previousModelValue, row.Column)
		}
		for _, row := range schemas {
			v := m.GetFieldValue(modelValue, row.Column)
			if IsEmpty(v) {
				continue
			}
			if previousValues[row.Column] != v {
				updates[row.Column] = v
			}
		}
		if len(updates) > 0 {
			if errTx = tx.Model(model).
				Where(map[string]any{m.primaryKey: primaryKey}).
				Updates(updates).Error; errTx != nil {
				return errTx
			}
			diffAttrs = make([]*DiffAttr, 0, len(schemas))
			for _, row := range schemas {
				v := m.GetFieldValue(modelValue, row.Column)
				if previousValues[row.Column] != v {
					diffAttrs = append(diffAttrs, &DiffAttr{
						Column:   row.Column,
						Label:    row.Label,
						Previous: previousValues[row.Column],
						Current:  v,
					})
				}
			}
		}
		return
	})
	if err != nil {
		return nil, err
	}
	if au, ok := any(&model).(AfterUpdated); ok {
		au.AfterUpdated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := any(&model).(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model[T]) Delete(ctx context.Context, primaryKeyValue any) (err error) {
	if !m.HasScenario(schema.ScenarioDelete) {
		return ErrPermissionDenied
	}
	var model T
	err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		errTx = tx.Model(model).Delete(map[string]any{
			m.primaryKey: primaryKeyValue,
		}).Error
		if errTx != nil {
			return errTx
		}

		return nil
	})
	if err != nil {
		return
	}
	if ad, ok := any(&model).(AfterDeleted); ok {
		ad.AfterDeleted(ctx, m.GetDB())
	}
	return
}

func (m *Model[T]) Detail(ctx context.Context, primaryKey any) (model *T, err error) {
	if !m.HasScenario(schema.ScenarioDetail) {
		return model, ErrPermissionDenied
	}
	var (
		modelValue T
	)
	if err = m.GetDB().WithContext(ctx).Where(map[string]any{
		m.primaryKey: primaryKey,
	}).First(&modelValue).Error; err != nil {
		return
	}
	return &modelValue, nil
}

func (m *Model[T]) List(ctx context.Context, offset, limit int, queryBuilder *query.Builder) ([]*T, error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return nil, ErrPermissionDenied
	}
	var (
		model T
		err   error
	)
	listBuilder := queryBuilder.Clone()
	if offset >= 0 {
		listBuilder.Offset(offset)
	}
	if limit > 0 {
		listBuilder.Limit(limit)
	}
	search := query.New(m.GetDB(), model, listBuilder)
	values := make([]*T, 0)
	if err = search.All(ctx, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (m *Model[T]) Count(ctx context.Context, queryBuilder *query.Builder) (int64, error) {
	var model T
	search := query.New(m.GetDB(), model, queryBuilder)
	return search.Count(ctx)
}

func (m *Model[T]) Paginate(ctx context.Context, page, size int, queryBuilder *query.Builder) (int64, []*T, error) {
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = 20
	}
	totalCount, err := m.Count(ctx, queryBuilder)
	if err != nil {
		return 0, nil, err
	}
	offset := page * size
	data, err := m.List(ctx, offset, size, queryBuilder)
	if err != nil {
		return 0, nil, err
	}
	return totalCount, data, nil
}

func (m *Model[T]) Cursor(ctx context.Context, cursor string, limit int, queryBuilder *query.Builder) (nextCursor string, hasMore bool, data []*T, err error) {
	offset, err := decodeCursor(cursor)
	if err != nil {
		return
	}
	if limit <= 0 {
		limit = 20
	}
	data, err = m.List(ctx, offset, limit+1, queryBuilder)
	if err != nil {
		return
	}
	hasMore = len(data) > limit
	if hasMore {
		data = data[:limit]
	}
	if hasMore {
		nextCursor = encodeCursor(offset + limit)
	}
	return
}

func NewModel[T any](opts ...Option) (v *Model[T], err error) {
	v = &Model[T]{
		opts: newOptions(opts...),
	}
	v.db = v.opts.db.Session(&gorm.Session{
		NewDB: true,
	}).Debug()
	var model T
	if err = v.db.Statement.Parse(&model); err != nil {
		return
	}
	for _, field := range v.db.Statement.Schema.Fields {
		if field.PrimaryKey {
			v.primaryKey = field.DBName
			break
		}
	}
	if err = v.db.AutoMigrate(model); err != nil {
		return
	}
	if v.naming.TableName, err = schema.AutoMigrate(v.db.Statement.Context, v.GetDB(), model, v.opts.moduleName); err != nil {
		return
	}
	singularizeTable := inflector.Singularize(v.naming.TableName)
	v.naming.Pluralize = inflector.Pluralize(v.naming.TableName)
	v.naming.Singular = singularizeTable
	v.naming.ModuleName = v.opts.moduleName
	return
}
