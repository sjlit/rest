package rest

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// Global any signatures (Before hooks return error, After hooks have no return value)
type BeforeCreateFunc func(ctx context.Context, db *gorm.DB, model any) error
type BeforeUpdateFunc func(ctx context.Context, db *gorm.DB, model any) error
type BeforeDeleteFunc func(ctx context.Context, db *gorm.DB, model any) error

type AfterCreateFunc func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr)
type AfterUpdateFunc func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr)
type AfterSavedFunc func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr)
type AfterDeleteFunc func(ctx context.Context, db *gorm.DB, model any)

// Local generic signatures
type BeforeCreateHook[T any] func(ctx context.Context, db *gorm.DB, model *T) error
type BeforeUpdateHook[T any] func(ctx context.Context, db *gorm.DB, model *T) error
type BeforeDeleteHook[T any] func(ctx context.Context, db *gorm.DB, model *T) error

type AfterCreateHook[T any] func(ctx context.Context, db *gorm.DB, model *T, diffAttrs []*DiffAttr)
type AfterUpdateHook[T any] func(ctx context.Context, db *gorm.DB, model *T, diffAttrs []*DiffAttr)
type AfterSavedHook[T any] func(ctx context.Context, db *gorm.DB, model *T, diffAttrs []*DiffAttr)
type AfterDeleteHook[T any] func(ctx context.Context, db *gorm.DB, model *T)

// Internal erased storage types
type erasedBeforeHookFunc func(ctx context.Context, db *gorm.DB, model any) error
type erasedAfterHookFunc func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr)
type erasedAfterDeleteHookFunc func(ctx context.Context, db *gorm.DB, model any)

type modelHooks struct {
	beforeCreate []erasedBeforeHookFunc
	afterCreate  []erasedAfterHookFunc
	beforeUpdate []erasedBeforeHookFunc
	afterUpdate  []erasedAfterHookFunc
	afterSaved   []erasedAfterHookFunc
	beforeDelete []erasedBeforeHookFunc
	afterDelete  []erasedAfterDeleteHookFunc
}

// Wrapper functions (generic -> erased)
func wrapBeforeHook[T any](fn func(ctx context.Context, db *gorm.DB, model *T) error) erasedBeforeHookFunc {
	return func(ctx context.Context, db *gorm.DB, model any) error {
		typed, ok := model.(*T)
		if !ok {
			return fmt.Errorf("hook type mismatch: expected *%T, got %T", *new(T), model)
		}
		return fn(ctx, db, typed)
	}
}

func wrapAfterHook[T any](fn func(ctx context.Context, db *gorm.DB, model *T, diffAttrs []*DiffAttr)) erasedAfterHookFunc {
	return func(ctx context.Context, db *gorm.DB, model any, diffAttrs []*DiffAttr) {
		typed, ok := model.(*T)
		if !ok {
			return
		}
		fn(ctx, db, typed, diffAttrs)
	}
}

func wrapAfterDeleteHook[T any](fn func(ctx context.Context, db *gorm.DB, model *T)) erasedAfterDeleteHookFunc {
	return func(ctx context.Context, db *gorm.DB, model any) {
		typed, ok := model.(*T)
		if !ok {
			return
		}
		fn(ctx, db, typed)
	}
}

// Global registry and registration functions
var (
	globalBeforeCreate []erasedBeforeHookFunc
	globalAfterCreate  []erasedAfterHookFunc
	globalBeforeUpdate []erasedBeforeHookFunc
	globalAfterUpdate  []erasedAfterHookFunc
	globalAfterSaved   []erasedAfterHookFunc
	globalBeforeDelete []erasedBeforeHookFunc
	globalAfterDelete  []erasedAfterDeleteHookFunc
	globalMu           sync.RWMutex
)

func RegisterBeforeCreate(fn BeforeCreateFunc) {
	globalMu.Lock()
	globalBeforeCreate = append(globalBeforeCreate, erasedBeforeHookFunc(fn))
	globalMu.Unlock()
}

func RegisterAfterCreate(fn AfterCreateFunc) {
	globalMu.Lock()
	globalAfterCreate = append(globalAfterCreate, erasedAfterHookFunc(fn))
	globalMu.Unlock()
}

func RegisterBeforeUpdate(fn BeforeUpdateFunc) {
	globalMu.Lock()
	globalBeforeUpdate = append(globalBeforeUpdate, erasedBeforeHookFunc(fn))
	globalMu.Unlock()
}

func RegisterAfterUpdate(fn AfterUpdateFunc) {
	globalMu.Lock()
	globalAfterUpdate = append(globalAfterUpdate, erasedAfterHookFunc(fn))
	globalMu.Unlock()
}

func RegisterAfterSaved(fn AfterSavedFunc) {
	globalMu.Lock()
	globalAfterSaved = append(globalAfterSaved, erasedAfterHookFunc(fn))
	globalMu.Unlock()
}

func RegisterBeforeDelete(fn BeforeDeleteFunc) {
	globalMu.Lock()
	globalBeforeDelete = append(globalBeforeDelete, erasedBeforeHookFunc(fn))
	globalMu.Unlock()
}

func RegisterAfterDelete(fn AfterDeleteFunc) {
	globalMu.Lock()
	globalAfterDelete = append(globalAfterDelete, erasedAfterDeleteHookFunc(fn))
	globalMu.Unlock()
}
