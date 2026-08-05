package rest

import (
	"context"

	"git.nobla.cn/golang/rest/v3/query"
)

// TypedModel 是编译期类型安全的模型包装, 内部委托给动态核心 Model。
// 若模型类型在运行时才确定(批量实例化、动态发现), 请使用 Model。
type TypedModel[T any] struct {
	*Model
}

func NewTypedModel[T any](opts ...Option) (v *TypedModel[T], err error) {
	var model T
	core, err := NewModel(model, opts...)
	if err != nil {
		return nil, err
	}
	return &TypedModel[T]{Model: core}, nil
}

func (m *TypedModel[T]) Create(ctx context.Context, model *T) (diffAttrs []*DiffAttr, err error) {
	return m.Model.Create(ctx, model)
}

func (m *TypedModel[T]) Update(ctx context.Context, primaryKey any, model *T, columns ...string) (primaryKeyValue any, err error) {
	return m.Model.Update(ctx, primaryKey, model, columns...)
}

func (m *TypedModel[T]) Delete(ctx context.Context, primaryKey any) (primaryKeyValue any, err error) {
	return m.Model.Delete(ctx, primaryKey)
}

func (m *TypedModel[T]) Detail(ctx context.Context, primaryKey any) (model *T, err error) {
	v, err := m.Model.Detail(ctx, primaryKey)
	if err != nil {
		return nil, err
	}
	return v.(*T), nil
}

func (m *TypedModel[T]) List(ctx context.Context, offset, limit int, queryBuilder *query.Builder) ([]*T, error) {
	v, err := m.Model.List(ctx, offset, limit, queryBuilder)
	if err != nil {
		return nil, err
	}
	return v.([]*T), nil
}

func (m *TypedModel[T]) Paginate(ctx context.Context, page, size int, queryBuilder *query.Builder) (int64, []*T, error) {
	total, v, err := m.Model.Paginate(ctx, page, size, queryBuilder)
	if err != nil {
		return 0, nil, err
	}
	return total, v.([]*T), nil
}

func (m *TypedModel[T]) Cursor(ctx context.Context, cursor string, limit int, queryBuilder *query.Builder) (nextCursor string, hasMore bool, data []*T, err error) {
	nextCursor, hasMore, v, err := m.Model.Cursor(ctx, cursor, limit, queryBuilder)
	if err != nil {
		return "", false, nil, err
	}
	return nextCursor, hasMore, v.([]*T), nil
}

func (m *TypedModel[T]) RegisterBeforeCreate(fn BeforeCreateHook[T]) {
	m.Model.RegisterBeforeCreate(BeforeCreateFunc(wrapBeforeHook(fn)))
}

func (m *TypedModel[T]) RegisterAfterCreate(fn AfterCreateHook[T]) {
	m.Model.RegisterAfterCreate(AfterCreateFunc(wrapAfterHook(fn)))
}

func (m *TypedModel[T]) RegisterBeforeUpdate(fn BeforeUpdateHook[T]) {
	m.Model.RegisterBeforeUpdate(BeforeUpdateFunc(wrapBeforeHook(fn)))
}

func (m *TypedModel[T]) RegisterAfterUpdate(fn AfterUpdateHook[T]) {
	m.Model.RegisterAfterUpdate(AfterUpdateFunc(wrapAfterHook(fn)))
}

func (m *TypedModel[T]) RegisterAfterSaved(fn AfterSavedHook[T]) {
	m.Model.RegisterAfterSaved(AfterSavedFunc(wrapAfterHook(fn)))
}

func (m *TypedModel[T]) RegisterBeforeDelete(fn BeforeDeleteHook[T]) {
	m.Model.RegisterBeforeDelete(BeforeDeleteFunc(wrapBeforeHook(fn)))
}

func (m *TypedModel[T]) RegisterAfterDelete(fn AfterDeleteHook[T]) {
	m.Model.RegisterAfterDelete(AfterDeleteFunc(wrapAfterDeleteHook(fn)))
}
