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
	db          *gorm.DB
	opts        *options
	naming      Naming
	primaryKey  string
	globalHooks *modelHooks // NewModel 时从全局注册表快照
	localHooks  *modelHooks // 实例级追加
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

func (m *Model[T]) OpenAPIEnabled() bool {
	return m.opts.enableOpenAPI
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

func (m *Model[T]) SetFieldValue(stmt *gorm.Statement, refValue reflect.Value, column string, value any) {
	var (
		rawField *gormSchema.Field
	)
	refVal := reflect.Indirect(refValue)
	for _, field := range stmt.Schema.Fields {
		if field.DBName == column || field.Name == column {
			rawField = field
			break
		}
	}
	if rawField == nil {
		return
	}
	var targetValue reflect.Value
	targetValue = refVal
	for _, i := range rawField.StructField.Index {
		targetValue = targetValue.Field(i)
	}
	targetValue.Set(reflect.ValueOf(value))
}

func (m *Model[T]) RegisterBeforeCreate(fn BeforeCreateHook[T]) {
	m.initLocalHooks()
	m.localHooks.beforeCreate = append(m.localHooks.beforeCreate, wrapBeforeHook(fn))
}

func (m *Model[T]) RegisterAfterCreate(fn AfterCreateHook[T]) {
	m.initLocalHooks()
	m.localHooks.afterCreate = append(m.localHooks.afterCreate, wrapAfterHook(fn))
}

func (m *Model[T]) RegisterBeforeUpdate(fn BeforeUpdateHook[T]) {
	m.initLocalHooks()
	m.localHooks.beforeUpdate = append(m.localHooks.beforeUpdate, wrapBeforeHook(fn))
}

func (m *Model[T]) RegisterAfterUpdate(fn AfterUpdateHook[T]) {
	m.initLocalHooks()
	m.localHooks.afterUpdate = append(m.localHooks.afterUpdate, wrapAfterHook(fn))
}

func (m *Model[T]) RegisterAfterSaved(fn AfterSavedHook[T]) {
	m.initLocalHooks()
	m.localHooks.afterSaved = append(m.localHooks.afterSaved, wrapAfterHook(fn))
}

func (m *Model[T]) RegisterBeforeDelete(fn BeforeDeleteHook[T]) {
	m.initLocalHooks()
	m.localHooks.beforeDelete = append(m.localHooks.beforeDelete, wrapBeforeHook(fn))
}

func (m *Model[T]) RegisterAfterDelete(fn AfterDeleteHook[T]) {
	m.initLocalHooks()
	m.localHooks.afterDelete = append(m.localHooks.afterDelete, wrapAfterDeleteHook(fn))
}

func (m *Model[T]) initLocalHooks() {
	if m.localHooks == nil {
		m.localHooks = &modelHooks{}
	}
}

func (m *Model[T]) runBeforeHooks(
	ctx context.Context,
	db *gorm.DB,
	model *T,
	globalFns, localFns []erasedBeforeHookFunc,
) error {
	for _, fns := range [][]erasedBeforeHookFunc{globalFns, localFns} {
		for _, fn := range fns {
			if err := fn(ctx, db, model); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Model[T]) runAfterHooks(
	ctx context.Context,
	db *gorm.DB,
	model *T,
	diffAttrs []*DiffAttr,
	globalFns, localFns []erasedAfterHookFunc,
) {
	for _, fns := range [][]erasedAfterHookFunc{globalFns, localFns} {
		for _, fn := range fns {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 记录 panic，不阻断主流程
					}
				}()
				fn(ctx, db, model, diffAttrs)
			}()
		}
	}
}

func (m *Model[T]) runAfterDeleteHooks(
	ctx context.Context,
	db *gorm.DB,
	model *T,
	globalFns, localFns []erasedAfterDeleteHookFunc,
) {
	for _, fns := range [][]erasedAfterDeleteHookFunc{globalFns, localFns} {
		for _, fn := range fns {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 记录 panic，不阻断主流程
					}
				}()
				fn(ctx, db, model)
			}()
		}
	}
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
	} else {
		runtimeScope = &RuntimeScope{
			Schemas:    schemas,
			ModuleName: m.GetNaming().ModuleName,
			TableName:  m.GetNaming().TableName,
			Scenario:   schema.ScenarioCreate,
		}
		ctx = WithRuntimeScope(ctx, runtimeScope)
	}
	if err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		// BeforeCreate hooks
		if errTx = m.runBeforeHooks(ctx, tx, model,
			m.globalHooks.beforeCreate, m.localHooks.beforeCreate); errTx != nil {
			return
		}

		if errTx = tx.Create(model).Error; errTx != nil {
			return
		}
		diffAttrs = make([]*DiffAttr, 0, len(schemas))
		modelRef := reflect.ValueOf(model)
		for _, row := range schemas {
			diffAttrs = append(diffAttrs, &DiffAttr{
				Column:   row.Column,
				Label:    row.Label,
				Previous: nil,
				Current:  m.GetFieldValue(modelRef, row.Column),
			})
		}

		return
	}); err != nil {
		return nil, err
	}

	// AfterCreate hooks
	m.runAfterHooks(ctx, m.GetDB(), model, diffAttrs,
		m.globalHooks.afterCreate, m.localHooks.afterCreate)

	// AfterSaved hooks
	m.runAfterHooks(ctx, m.GetDB(), model, diffAttrs,
		m.globalHooks.afterSaved, m.localHooks.afterSaved)

	// 兼容现有接口式 hook
	if ac, ok := any(model).(AfterCreated); ok {
		ac.AfterCreated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := any(model).(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model[T]) Update(ctx context.Context, primaryKey any, model *T, columns ...string) (primaryKeyValue any, err error) {
	if !m.HasScenario(schema.ScenarioUpdate) {
		return nil, ErrPermissionDenied
	}
	var (
		updates        map[string]any
		previousValues map[string]any
		schemas        []schema.Schema
		diffAttrs      []*DiffAttr
	)
	if schemas, err = schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioUpdate); err != nil {
		return nil, err
	}
	modelRef := reflect.ValueOf(model)
	updates = make(map[string]any)
	previousValues = make(map[string]any)
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope != nil {
		runtimeScope.Schemas = schemas
	} else {
		runtimeScope = &RuntimeScope{
			Schemas:    schemas,
			ModuleName: m.GetNaming().ModuleName,
			TableName:  m.GetNaming().TableName,
			Scenario:   schema.ScenarioUpdate,
		}
		ctx = WithRuntimeScope(ctx, runtimeScope)
	}
	err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		previousModel := new(T)
		if errTx = tx.Where(map[string]any{m.primaryKey: primaryKey}).First(previousModel).Error; errTx != nil {
			return errTx
		}
		previousModelRef := reflect.ValueOf(previousModel)
		primaryKeyValue = m.GetFieldValue(previousModelRef, m.primaryKey)
		for _, row := range schemas {
			previousValues[row.Column] = m.GetFieldValue(previousModelRef, row.Column)
		}
		for _, row := range schemas {
			if (len(columns) == 0 || slices.Contains(columns, row.Column)) && row.PrimaryKey == 0 {
				v := m.GetFieldValue(modelRef, row.Column)
				if previousValues[row.Column] != v {
					updates[row.Column] = v
				}
			}
		}
		if len(updates) > 0 {
			if errTx = m.runBeforeHooks(ctx, tx, model, m.globalHooks.beforeUpdate, m.localHooks.beforeUpdate); errTx != nil {
				return errTx
			}
			if errTx = tx.Model(previousModel).
				Where(map[string]any{m.primaryKey: primaryKey}).
				Updates(updates).Error; errTx != nil {
				return errTx
			}
			if errTx = tx.Model(model).Where(map[string]any{m.primaryKey: primaryKey}).First(model).Error; errTx != nil {
				return errTx
			}
			diffAttrs = make([]*DiffAttr, 0, len(schemas))
			for _, row := range schemas {
				v := m.GetFieldValue(modelRef, row.Column)
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
	m.runAfterHooks(ctx, m.GetDB(), model, diffAttrs,
		m.globalHooks.afterUpdate, m.localHooks.afterUpdate)
	m.runAfterHooks(ctx, m.GetDB(), model, diffAttrs,
		m.globalHooks.afterSaved, m.localHooks.afterSaved)
	if au, ok := any(model).(AfterUpdated); ok {
		au.AfterUpdated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := any(model).(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model[T]) Delete(ctx context.Context, primaryKey any) (primaryKeyValue any, err error) {
	if !m.HasScenario(schema.ScenarioDelete) {
		return nil, ErrPermissionDenied
	}
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope == nil {
		runtimeScope = &RuntimeScope{
			ModuleName: m.GetNaming().ModuleName,
			TableName:  m.GetNaming().TableName,
			Scenario:   schema.ScenarioDelete,
		}
		ctx = WithRuntimeScope(ctx, runtimeScope)
	}
	model := new(T)
	err = m.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) (errTx error) {
		// 先查询完整记录
		if errTx = tx.Where(map[string]any{m.primaryKey: primaryKey}).First(model).Error; errTx != nil {
			return errTx
		}
		primaryKeyValue = m.GetFieldValue(reflect.ValueOf(model), m.primaryKey)

		// BeforeDelete hooks
		if errTx = m.runBeforeHooks(ctx, tx, model,
			m.globalHooks.beforeDelete, m.localHooks.beforeDelete); errTx != nil {
			return errTx
		}

		errTx = tx.Delete(model).Error
		return
	})
	if err != nil {
		return
	}

	// AfterDelete hooks
	m.runAfterDeleteHooks(ctx, m.GetDB(), model, m.globalHooks.afterDelete, m.localHooks.afterDelete)
	// 兼容现有接口式 hook
	if ad, ok := any(model).(AfterDeleted); ok {
		ad.AfterDeleted(ctx, m.GetDB())
	}
	return
}

func (m *Model[T]) Detail(ctx context.Context, primaryKey any) (model *T, err error) {
	if !m.HasScenario(schema.ScenarioDetail) {
		return model, ErrPermissionDenied
	}
	model = new(T)
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope == nil {
		runtimeScope = &RuntimeScope{
			ModuleName: m.GetNaming().ModuleName,
			TableName:  m.GetNaming().TableName,
			Scenario:   schema.ScenarioDetail,
		}
		ctx = WithRuntimeScope(ctx, runtimeScope)
	}
	if schemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioDetail); err == nil {
		runtimeScope.Schemas = schemas
	}
	if err = m.applyPreloads(ctx, m.GetDB().WithContext(ctx), runtimeScope.Schemas, schema.ScenarioDetail, make(map[string]bool), "").Where(map[string]any{
		m.primaryKey: primaryKey,
	}).First(model).Error; err != nil {
		return
	}
	return model, nil
}

func (m *Model[T]) List(ctx context.Context, offset, limit int, queryBuilder *query.Builder) ([]*T, error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return nil, ErrPermissionDenied
	}
	var (
		model T
		err   error
	)
	listBuilder := query.NewBuilder()
	if queryBuilder != nil {
		listBuilder = queryBuilder.Clone()
	}
	if offset >= 0 {
		listBuilder.Offset(offset)
	}
	if limit > 0 {
		listBuilder.Limit(limit)
	}
	runtimeScope := RuntimeScopeFromContext(ctx)
	if runtimeScope == nil {
		runtimeScope = &RuntimeScope{
			ModuleName: m.GetNaming().ModuleName,
			TableName:  m.GetNaming().TableName,
			Scenario:   schema.ScenarioList,
		}
		ctx = WithRuntimeScope(ctx, runtimeScope)
	}
	if schemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), m.naming.ModuleName, m.naming.TableName, schema.ScenarioList); err == nil {
		runtimeScope.Schemas = schemas
	}
	searchDB := m.applyPreloads(ctx, m.GetDB().WithContext(ctx), runtimeScope.Schemas, schema.ScenarioList, make(map[string]bool), "")
	search := query.New(searchDB, model, listBuilder)
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

func (m *Model[T]) applyPreloads(ctx context.Context, db *gorm.DB, schemas []schema.Schema, scenario string, visited map[string]bool, prefix string) *gorm.DB {
	for _, s := range schemas {
		if s.Relations.Type == "" {
			continue
		}
		path := s.Relations.Name
		if prefix != "" {
			path = prefix + "." + path
		}
		db = db.Preload(path)

		// 循环引用检测
		key := s.Relations.Module + ":" + s.Relations.Table + ":" + path
		if visited[key] {
			continue
		}
		visited[key] = true

		// 加载关联模型的 Schema 并递归
		if s.Relations.Module != "" && s.Relations.Table != "" {
			assocSchemas, err := schema.GetVisibleSchemas(ctx, m.GetDB(), s.Relations.Module, s.Relations.Table, scenario)
			if err == nil && len(assocSchemas) > 0 {
				db = m.applyPreloads(ctx, db, assocSchemas, scenario, visited, path)
			}
		}
	}
	return db
}

func NewModel[T any](opts ...Option) (v *Model[T], err error) {
	v = &Model[T]{
		opts: newOptions(opts...),
	}
	v.db = v.opts.db.Session(&gorm.Session{
		NewDB: true,
	})
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

	// 初始化局部 hooks（避免 nil pointer）
	v.localHooks = &modelHooks{}

	// 快照：复制全局 hooks 到实例
	globalMu.RLock()
	v.globalHooks = &modelHooks{
		beforeCreate: append([]erasedBeforeHookFunc(nil), globalBeforeCreate...),
		afterCreate:  append([]erasedAfterHookFunc(nil), globalAfterCreate...),
		beforeUpdate: append([]erasedBeforeHookFunc(nil), globalBeforeUpdate...),
		afterUpdate:  append([]erasedAfterHookFunc(nil), globalAfterUpdate...),
		afterSaved:   append([]erasedAfterHookFunc(nil), globalAfterSaved...),
		beforeDelete: append([]erasedBeforeHookFunc(nil), globalBeforeDelete...),
		afterDelete:  append([]erasedAfterDeleteHookFunc(nil), globalAfterDelete...),
	}
	globalMu.RUnlock()

	return
}
