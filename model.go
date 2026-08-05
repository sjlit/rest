package rest

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/sjlit/rest/v3/internal/inflector"
	"github.com/sjlit/rest/v3/internal/safelog"
	"github.com/sjlit/rest/v3/query"
	"github.com/sjlit/rest/v3/schema"
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

// Model 是模型类型在运行时才确定的动态模型, 支持把任意结构体(实例或 reflect.Type)交给它管理。
// 若希望获得编译期类型安全, 请使用 TypedModel[T]。
type Model struct {
	db          *gorm.DB
	opts        *options
	naming      Naming
	primaryKey  string
	modelType   reflect.Type // 归一化后的结构体类型
	globalHooks *modelHooks  // NewModel 时从全局注册表快照
	localHooks  *modelHooks  // 实例级追加
}

// normalizeModelType 将实例(值或指针)或 reflect.Type 归一化为结构体类型
func normalizeModelType(model any) (reflect.Type, error) {
	if model == nil {
		return nil, fmt.Errorf("rest: model is nil")
	}
	t, ok := model.(reflect.Type)
	if !ok {
		t = reflect.TypeOf(model)
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("rest: model must be a struct, got %s", t.Kind())
	}
	return t, nil
}

// checkModelType 校验调用方传入的模型实例与 modelType 一致
func (m *Model) checkModelType(model any) error {
	if model == nil {
		return fmt.Errorf("rest: model is nil")
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t != m.modelType {
		return fmt.Errorf("rest: model type mismatch: expected %s, got %s", m.modelType, t)
	}
	return nil
}

func (m *Model) GetDB() *gorm.DB {
	db := m.db.Session(&gorm.Session{NewDB: true})
	instance := reflect.New(m.modelType).Interface()
	db = db.Model(instance)
	// Model() 不会触发 Schema 解析, 显式 Parse 以保证 Statement.Schema 可用
	_ = db.Statement.Parse(instance)
	return db
}

func (m *Model) HasScenario(s string) bool {
	if len(m.opts.scenarios) == 0 {
		return true
	}
	return slices.Contains(m.opts.scenarios, s)
}

func (m *Model) GetPrimaryKey() string {
	return m.primaryKey
}

func (m *Model) OpenAPIEnabled() bool {
	return m.opts.enableOpenAPI
}

func (m *Model) GetNaming() Naming {
	return m.naming
}

func (m *Model) ModelType() reflect.Type {
	return m.modelType
}

func (m *Model) GetFields() []*gormSchema.Field {
	return m.GetDB().Statement.Schema.Fields
}

func (m *Model) GetFieldValue(refValue reflect.Value, column string) any {
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

func (m *Model) SetFieldValue(stmt *gorm.Statement, refValue reflect.Value, column string, value any) {
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

func (m *Model) RegisterBeforeCreate(fn BeforeCreateFunc) {
	m.initLocalHooks()
	m.localHooks.beforeCreate = append(m.localHooks.beforeCreate, erasedBeforeHookFunc(fn))
}

func (m *Model) RegisterAfterCreate(fn AfterCreateFunc) {
	m.initLocalHooks()
	m.localHooks.afterCreate = append(m.localHooks.afterCreate, erasedAfterHookFunc(fn))
}

func (m *Model) RegisterBeforeUpdate(fn BeforeUpdateFunc) {
	m.initLocalHooks()
	m.localHooks.beforeUpdate = append(m.localHooks.beforeUpdate, erasedBeforeHookFunc(fn))
}

func (m *Model) RegisterAfterUpdate(fn AfterUpdateFunc) {
	m.initLocalHooks()
	m.localHooks.afterUpdate = append(m.localHooks.afterUpdate, erasedAfterHookFunc(fn))
}

func (m *Model) RegisterAfterSaved(fn AfterSavedFunc) {
	m.initLocalHooks()
	m.localHooks.afterSaved = append(m.localHooks.afterSaved, erasedAfterHookFunc(fn))
}

func (m *Model) RegisterBeforeDelete(fn BeforeDeleteFunc) {
	m.initLocalHooks()
	m.localHooks.beforeDelete = append(m.localHooks.beforeDelete, erasedBeforeHookFunc(fn))
}

func (m *Model) RegisterAfterDelete(fn AfterDeleteFunc) {
	m.initLocalHooks()
	m.localHooks.afterDelete = append(m.localHooks.afterDelete, erasedAfterDeleteHookFunc(fn))
}

func (m *Model) initLocalHooks() {
	if m.localHooks == nil {
		m.localHooks = &modelHooks{}
	}
}

func (m *Model) runBeforeHooks(
	ctx context.Context,
	db *gorm.DB,
	model any,
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

func (m *Model) runAfterHooks(
	ctx context.Context,
	db *gorm.DB,
	model any,
	diffAttrs []*DiffAttr,
	globalFns, localFns []erasedAfterHookFunc,
) {
	for _, fns := range [][]erasedAfterHookFunc{globalFns, localFns} {
		for _, fn := range fns {
			safelog.SafeRun("after", func() {
				fn(ctx, db, model, diffAttrs)
			})
		}
	}
}

func (m *Model) runAfterDeleteHooks(
	ctx context.Context,
	db *gorm.DB,
	model any,
	globalFns, localFns []erasedAfterDeleteHookFunc,
) {
	for _, fns := range [][]erasedAfterDeleteHookFunc{globalFns, localFns} {
		for _, fn := range fns {
			safelog.SafeRun("afterDelete", func() {
				fn(ctx, db, model)
			})
		}
	}
}

func (m *Model) Create(ctx context.Context, model any) (diffAttrs []*DiffAttr, err error) {
	if !m.HasScenario(schema.ScenarioCreate) {
		return nil, ErrPermissionDenied
	}
	if err = m.checkModelType(model); err != nil {
		return nil, err
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
	if ac, ok := model.(AfterCreated); ok {
		ac.AfterCreated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := model.(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model) Update(ctx context.Context, primaryKey any, model any, columns ...string) (primaryKeyValue any, err error) {
	if !m.HasScenario(schema.ScenarioUpdate) {
		return nil, ErrPermissionDenied
	}
	if err = m.checkModelType(model); err != nil {
		return nil, err
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
		previousModel := reflect.New(m.modelType).Interface()
		if errTx = tx.Where(map[string]any{m.primaryKey: primaryKey}).First(previousModel).Error; errTx != nil {
			return errTx
		}
		previousModelRef := reflect.ValueOf(previousModel)
		primaryKeyValue = m.GetFieldValue(previousModelRef, m.primaryKey)
		for _, row := range schemas {
			previousValues[row.Column] = m.GetFieldValue(previousModelRef, row.Column)
		}
		for _, row := range schemas {
			// Skip primary key and fields marked as disabled by GORM (e.g. <-:create reverse relations).
			// This prevents callers from bypassing Schema filtering via the columns list.
			if row.PrimaryKey == 0 && !slices.Contains(row.Attributes.Disable, schema.ScenarioUpdate) {
				if len(columns) == 0 || slices.Contains(columns, row.Column) {
					v := m.GetFieldValue(modelRef, row.Column)
					if previousValues[row.Column] != v {
						updates[row.Column] = v
					}
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
	if au, ok := model.(AfterUpdated); ok {
		au.AfterUpdated(ctx, m.GetDB(), diffAttrs)
	}
	if as, ok := model.(AfterSaved); ok {
		as.AfterSaved(ctx, m.GetDB(), diffAttrs)
	}
	return
}

func (m *Model) Delete(ctx context.Context, primaryKey any) (primaryKeyValue any, err error) {
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
	model := reflect.New(m.modelType).Interface()
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
	if ad, ok := model.(AfterDeleted); ok {
		ad.AfterDeleted(ctx, m.GetDB())
	}
	return
}

func (m *Model) Detail(ctx context.Context, primaryKey any) (model any, err error) {
	if !m.HasScenario(schema.ScenarioDetail) {
		return model, ErrPermissionDenied
	}
	model = reflect.New(m.modelType).Interface()
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

func (m *Model) List(ctx context.Context, offset, limit int, queryBuilder *query.Builder) (any, error) {
	if !m.HasScenario(schema.ScenarioSearch) {
		return nil, ErrPermissionDenied
	}
	var (
		model any
		err   error
	)
	model = reflect.New(m.modelType).Elem().Interface()
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
	values := reflect.New(reflect.SliceOf(reflect.PointerTo(m.modelType)))
	if err = search.All(ctx, values.Interface()); err != nil {
		return nil, err
	}
	return values.Elem().Interface(), nil
}

func (m *Model) Count(ctx context.Context, queryBuilder *query.Builder) (int64, error) {
	if queryBuilder == nil {
		queryBuilder = query.NewBuilder()
	}
	model := reflect.New(m.modelType).Elem().Interface()
	search := query.New(m.GetDB(), model, queryBuilder)
	return search.Count(ctx)
}

func (m *Model) Paginate(ctx context.Context, page, size int, queryBuilder *query.Builder) (int64, any, error) {
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

func (m *Model) Cursor(ctx context.Context, cursor string, limit int, queryBuilder *query.Builder) (nextCursor string, hasMore bool, data any, err error) {
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
	rv := reflect.ValueOf(data)
	hasMore = rv.Len() > limit
	if hasMore {
		data = rv.Slice(0, limit).Interface()
		nextCursor = encodeCursor(offset + limit)
	}
	return
}

func (m *Model) applyPreloads(ctx context.Context, db *gorm.DB, schemas []schema.Schema, scenario string, visited map[string]bool, prefix string) *gorm.DB {
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

// NewModel 为任意结构体(实例或 reflect.Type)创建动态模型
func NewModel(model any, opts ...Option) (v *Model, err error) {
	var modelType reflect.Type
	if modelType, err = normalizeModelType(model); err != nil {
		return nil, err
	}
	v = &Model{
		opts:      newOptions(opts...),
		modelType: modelType,
	}
	v.db = v.opts.db.Session(&gorm.Session{
		NewDB: true,
	})
	instance := reflect.New(modelType).Interface()
	if v.opts.moduleName == "" {
		if mm, ok := instance.(ModuleNamer); ok {
			v.opts.moduleName = mm.ModuleName()
		}
	}
	if len(v.opts.scenarios) == 0 {
		if sp, ok := instance.(ScenarioProvider); ok {
			v.opts.scenarios = sp.Scenarios()
		}
	}
	if err = v.db.Statement.Parse(instance); err != nil {
		return
	}
	for _, field := range v.db.Statement.Schema.Fields {
		if field.PrimaryKey {
			v.primaryKey = field.DBName
			break
		}
	}
	if err = v.db.AutoMigrate(instance); err != nil {
		return
	}
	if v.naming.TableName, err = schema.AutoMigrate(v.db.Statement.Context, v.GetDB(), instance, v.opts.moduleName); err != nil {
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
