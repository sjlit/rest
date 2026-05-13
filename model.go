package rest

import (
	"context"
	"reflect"
	"slices"

	"git.nobla.cn/golang/rest/internal/inflector"
	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
	gormSchema "gorm.io/gorm/schema"
)

type Model[T any] struct {
	ctx        context.Context
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
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName: m.naming.ModuleName,
		TableName:  m.naming.TableName,
		Scenario:   schema.ScenarioCreate,
		Schemas:    schemas,
		Context:    m.ctx,
	})
	if err = m.GetDB().WithContext(childCtx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(model).Error
	}); err != nil {
		return nil, err
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
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName:      m.naming.ModuleName,
		TableName:       m.naming.TableName,
		Scenario:        schema.ScenarioUpdate,
		Schemas:         schemas,
		PrimaryKeyValue: primaryKey,
		Context:         m.ctx,
	})
	err = m.GetDB().WithContext(childCtx).Transaction(func(tx *gorm.DB) error {
		previousModel := reflect.New(reflect.Indirect(reflect.ValueOf(model)).Type()).Interface()
		if errTx := tx.Where(map[string]any{m.primaryKey: primaryKey}).First(previousModel).Error; errTx != nil {
			err = errTx
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
			if errTx := tx.Model(model).
				Where(map[string]any{m.primaryKey: primaryKey}).
				Updates(updates).Error; errTx != nil {
				return errTx
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	diffAttrs = make([]*DiffAttr, 0, 10)
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
	return
}

func (m *Model[T]) Delete(ctx context.Context, primaryKeyValue any) (err error) {
	if !m.HasScenario(schema.ScenarioDelete) {
		return ErrPermissionDenied
	}
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName:      m.naming.ModuleName,
		TableName:       m.naming.TableName,
		Scenario:        schema.ScenarioDelete,
		PrimaryKeyValue: primaryKeyValue,
		Context:         m.ctx,
	})
	var model T
	err = m.GetDB().WithContext(childCtx).Transaction(func(tx *gorm.DB) error {
		errTx := tx.Model(model).Delete(map[string]any{
			m.primaryKey: primaryKeyValue,
		}).Error
		if errTx != nil {
			return errTx
		}
		return nil
	})
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

func (m *Model[T]) Find(ctx context.Context, offset, limit int, queryBuilder *query.Builder) (totalCount int64, values []*T, err error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return 0, nil, ErrPermissionDenied
	}
	var (
		model   T
		schemas []schema.Schema
	)
	if schemas, err = schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioList); err != nil {
		return
	}
	childCtx := WithRuntimeScope(ctx, &RuntimeScope{
		ModuleName: m.naming.ModuleName,
		TableName:  m.naming.TableName,
		Scenario:   schema.ScenarioList,
		Schemas:    schemas,
		Context:    m.ctx,
	})
	search := query.New(m.GetDB(), model, queryBuilder)
	search.Builder().Offset(0).Limit(0)
	if totalCount, err = search.
		Count(childCtx); err != nil {
		return
	}
	search.Builder().Offset(offset).Limit(limit)
	values = make([]*T, 0)
	if err = search.
		All(childCtx, &values); err != nil {
		return
	}
	return
}

func NewModel[T any](ctx context.Context, opts ...Option) (v *Model[T], err error) {
	v = &Model[T]{
		opts: newOptions(opts...),
		ctx:  ctx,
	}
	v.db = v.opts.db.Session(&gorm.Session{
		NewDB:   true,
		Context: ctx,
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
	if v.naming.TableName, err = schema.AutoMigrate(ctx, v.GetDB(), model, v.opts.moduleName); err != nil {
		return
	}
	singularizeTable := inflector.Singularize(v.naming.TableName)
	v.naming.Pluralize = inflector.Pluralize(v.naming.TableName)
	v.naming.Singular = singularizeTable
	v.naming.ModuleName = v.opts.moduleName
	return
}
