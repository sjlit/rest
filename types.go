package rest

import (
	"context"
	"net/http"

	"gorm.io/gorm"
)

const (
	QueryParamPage     = "page"
	QueryParamPageSize = "page_size"
	QueryParamSort     = "sort"
	QueryParamFormat   = "__format"
	QueryParamScenario = "scenario"
)

type (
	Naming struct {
		Pluralize  string
		Singular   string
		ModuleName string
		TableName  string
	}

	DiffAttr struct {
		Column   string `json:"column"`
		Label    string `json:"label"`
		Previous any    `json:"previous"`
		Current  any    `json:"current"`
	}
)

type (
	Router interface {
		Handle(method string, path string, handler http.HandlerFunc)
	}

	Responder interface {
		Respond(w http.ResponseWriter, r *http.Request, data any)
	}

	CreateResult struct {
		ID any `json:"id"`
	}

	UpdateResult struct {
		ID any `json:"id"`
	}

	DeletedResult struct {
		ID any `json:"id"`
	}

	PageResult struct {
		Page       int   `json:"page"`
		PageSize   int   `json:"page_size"`
		TotalCount int64 `json:"total_count"`
		Data       any   `json:"data"`
	}
)

type (
	BaseModel struct {
		ID        uint           `json:"id" gorm:"primarykey"`
		CreatedAt int64          `json:"created_at" gorm:"autoCreateTime"`
		UpdatedAt int64          `json:"updated_at" gorm:"autoUpdateTime"`
		DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	}

	TenantModel struct {
		BaseModel
		TenantID string `json:"tenant_id" gorm:"column:tenant_id;type:char(60);index"`
	}

	ModuleNamer interface {
		ModuleName() string
	}

	ScenarioProvider interface {
		Scenarios() []string
	}
)

type (
	// ResolveTenantFunc 获取租户信息
	ResolveTenantFunc func(ctx context.Context, r *http.Request) (string, error)

	// ResolveUserFunc 获取用户信息
	ResolveUserFunc func(ctx context.Context, r *http.Request) (string, error)
)

type (
	// 新建之后的回调, 不在新建的事务内,是真正保存到数据库以后才执行的回调
	AfterCreated interface {
		AfterCreated(ctx context.Context, tx *gorm.DB, diff []*DiffAttr)
	}

	// 更新之后的回调, 不在更新的事务内,是真正保存到数据库以后才执行的回调
	AfterUpdated interface {
		AfterUpdated(ctx context.Context, tx *gorm.DB, diff []*DiffAttr)
	}

	// 保存之后的回调, 不在保存的事务内,是真正保存到数据库以后才执行的回调
	AfterDeleted interface {
		AfterDeleted(ctx context.Context, tx *gorm.DB)
	}

	// 删除之后的回调, 不在删除的事务内,是真正保存到数据库以后才执行的回调
	AfterSaved interface {
		AfterSaved(ctx context.Context, tx *gorm.DB, diff []*DiffAttr)
	}
)

type (
	TypeValue[T any] struct {
		Label string `json:"label"`
		Value T      `json:"value"`
	}

	TierValue[T comparable] struct {
		Label    string          `json:"label"`
		Value    T               `json:"value"`
		Parent   T               `json:"-"`
		Used     bool            `json:"-"`
		Children []*TierValue[T] `json:"children,omitempty"`
	}
)
