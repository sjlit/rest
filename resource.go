package rest

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sjlit/rest/v3/formats"
	"github.com/sjlit/rest/v3/openapi"
	"github.com/sjlit/rest/v3/query"
	"github.com/sjlit/rest/v3/schema"
	"gorm.io/gorm"
)

// Resource 是动态模型的 HTTP 资源包装, 自动生成标准 RESTful 路由。
// 若希望获得编译期类型安全, 请使用 TypedResource[T]。
type Resource struct {
	model         *Model
	prefix        string
	router        Router
	responder     Responder
	registered    atomic.Bool
	formatter     *formats.Formatter
	tenantResolve ResolveTenantFunc
	userResolve   ResolveUserFunc
}

type ResourceConfig struct {
	Router        Router
	Responder     Responder
	Formatter     *formats.Formatter
	Prefix        string
	TenantResolve ResolveTenantFunc
	UserResolve   ResolveUserFunc
}

func (r *Resource) BuildUri(scenario string) (method string, uri string) {
	return r.buildUri(scenario)
}

func (r *Resource) buildUri(scenario string) (method string, uri string) {
	switch scenario {
	case schema.ScenarioCreate:
		method = http.MethodPost
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular)
	case schema.ScenarioUpdate:
		method = http.MethodPut
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, ":id")
	case schema.ScenarioDelete:
		method = http.MethodDelete
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, ":id")
	case schema.ScenarioSearch:
		method = http.MethodGet
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Pluralize)
	case schema.ScenarioDetail:
		method = http.MethodGet
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, "detail", ":id")
	case schema.ScenarioExport:
		method = http.MethodGet
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, "export")
	case "openapi":
		method = http.MethodGet
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, "openapi.json")
	}
	return
}

func (r *Resource) buildQuery(req *http.Request, schemas []schema.Schema) *query.Builder {
	var (
		formValue string
	)
	builder := query.NewBuilder()
	qs := req.URL.Query()
	for _, row := range schemas {
		if row.Native == 0 {
			continue
		}
		columnName := row.Column + "[]"
		if qs.Has(columnName) {
			if len(qs[columnName]) > 1 {
				builder.Where(row.Column, query.OpIn, qs[columnName])
				continue
			} else if len(qs[columnName]) == 1 {
				builder.Where(row.Column, query.OpEq, qs[columnName][0])
				continue
			}
		}
		formValue = qs.Get(row.Column)
		if formValue == "" {
			continue
		}
		switch row.Format {
		case schema.FormatString, schema.FormatText:
			if row.Attributes.Match == schema.MatchExactly {
				builder.Where(row.Column, query.OpEq, formValue)
			} else {
				builder.Where(row.Column, query.OpLike, formValue+"%")
			}
		case schema.FormatTime, schema.FormatDate, schema.FormatDatetime, schema.FormatTimestamp:
			var sep string
			seps := []byte{',', '/'}
			for _, s := range seps {
				if strings.IndexByte(formValue, s) > -1 {
					sep = string(s)
				}
			}
			// 仅在找到分隔符时按范围解析, 否则 Split("") 会按字符切分导致误判
			if sep != "" {
				if ss := strings.Split(formValue, sep); len(ss) == 2 {
					builder.WhereGroup(func(b *query.Builder) {
						b.Where(row.Column, query.OpGte, ss[0])
						b.Where(row.Column, query.OpLte, ss[1])
					})
					continue
				}
			}
			builder.Where(row.Column, query.OpEq, formValue)
		case schema.FormatInteger, schema.FormatFloat:
			builder.Where(row.Column, query.OpEq, formValue)
		default:
			if row.Type == schema.TypeString {
				if row.Attributes.Match == schema.MatchExactly {
					builder.Where(row.Column, query.OpEq, formValue)
				} else {
					builder.Where(row.Column, query.OpLike, formValue+"%")
				}
			} else {
				builder.Where(row.Column, query.OpEq, formValue)
			}
		}
	}
	sortPar := req.FormValue(QueryParamSort)
	if sortPar != "" {
		sorts := strings.SplitSeq(sortPar, ",")
		for s := range sorts {
			// 跳过空段(如 "name," 或 "a,,b"), 避免对空字符串取 s[0] 越界 panic
			if s == "" {
				continue
			}
			if s[0] == '-' {
				builder.OrderBy(s[1:], "DESC")
			} else {
				if s[0] == '+' {
					builder.OrderBy(s[1:], "ASC")
				} else {
					builder.OrderBy(s, "ASC")
				}
			}
		}
	}
	return builder
}

func (r *Resource) findPrimaryKey(req *http.Request, scenario string) string {
	_, fullUri := r.buildUri(scenario)
	// 1) 精确匹配（含 :id）：直接按段对比
	fullParts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(fullUri, "/"), "/"), "/")
	reqParts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(req.URL.Path, "/"), "/"), "/")
	if len(fullParts) == len(reqParts) {
		for i, part := range fullParts {
			if strings.HasPrefix(part, ":") {
				return reqParts[i]
			}
		}
	}
	// 2) 回退：去掉模板末尾的参数段后再做前缀截断，
	//    处理请求 path 比模板多一段(例如附加子路径)的边界情况。
	//    注意：不能用 fullUri 原串直接 CutPrefix，因为模板里的 ":id"
	//    是字面量，真实请求永远不会以 ":id" 结尾。
	prefix := fullUri
	if i := strings.LastIndex(prefix, "/:"); i >= 0 {
		prefix = prefix[:i]
	}
	if trimmed, ok := strings.CutPrefix(req.URL.Path, prefix); ok {
		trimmed = strings.TrimPrefix(trimmed, "/")
		if i := strings.IndexByte(trimmed, '/'); i >= 0 {
			trimmed = trimmed[:i]
		}
		return trimmed
	}
	return ""
}

func (r *Resource) getRuntimeScope(req *http.Request) (runtimeScope *RuntimeScope, err error) {
	runtimeScope = &RuntimeScope{
		ModuleName: r.model.GetNaming().ModuleName,
		TableName:  r.model.GetNaming().TableName,
		Context:    r.model.db.Statement.Context,
	}
	if r.userResolve != nil {
		if runtimeScope.User, err = r.userResolve(req.Context(), req); err != nil {
			return
		}
	}
	if r.tenantResolve != nil {
		if runtimeScope.TenantID, err = r.tenantResolve(req.Context(), req); err != nil {
			return
		}
	}
	return runtimeScope, nil
}

func (r *Resource) Register() {
	var (
		method string
		uri    string
	)
	if !r.registered.CompareAndSwap(false, true) {
		return
	}
	if r.router == nil || r.model == nil {
		return
	}
	if r.model.HasScenario(schema.ScenarioCreate) {
		method, uri = r.buildUri(schema.ScenarioCreate)
		r.router.Handle(method, uri, r.Create)
	}
	if r.model.HasScenario(schema.ScenarioUpdate) {
		method, uri = r.buildUri(schema.ScenarioUpdate)
		r.router.Handle(method, uri, r.Update)
	}
	if r.model.HasScenario(schema.ScenarioDelete) {
		method, uri = r.buildUri(schema.ScenarioDelete)
		r.router.Handle(method, uri, r.Delete)
	}
	if r.model.HasScenario(schema.ScenarioDetail) {
		method, uri = r.buildUri(schema.ScenarioDetail)
		r.router.Handle(method, uri, r.Detail)
	}
	if r.model.HasScenario(schema.ScenarioSearch) {
		method, uri = r.buildUri(schema.ScenarioSearch)
		r.router.Handle(method, uri, r.Search)
	}
	if r.model.HasScenario(schema.ScenarioExport) && r.formatter != nil {
		method, uri = r.buildUri(schema.ScenarioExport)
		r.router.Handle(method, uri, r.Export)
	}
	if r.model.OpenAPIEnabled() {
		method, uri = r.buildUri("openapi")
		r.router.Handle(method, uri, r.OpenApi)
	}
}

// httpStatusFor maps a framework Error to an HTTP status code.
// Unknown error values fall through to 500.
func httpStatusFor(err error) int {
	switch err {
	case ErrPermissionDenied:
		return http.StatusForbidden
	case ErrRecordNotFound:
		return http.StatusNotFound
	case ErrPayloadInvalid:
		return http.StatusBadRequest
	case ErrCreateFailed, ErrUpdateFailed, ErrDeleteFailed, ErrUnavailable:
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

func (r *Resource) Respond(res http.ResponseWriter, req *http.Request, data any) {
	if r.responder != nil {
		r.responder.Respond(res, req, data)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err, ok := data.(error); ok {
		res.WriteHeader(httpStatusFor(err))
		_ = json.NewEncoder(res).Encode(Error{
			Code:   errorCode(err),
			Reason: err.Error(),
		})
		return
	}
	json.NewEncoder(res).Encode(data)
}

// errorCode returns the framework Error code if err is one, otherwise 0.
func errorCode(err error) int {
	if e, ok := err.(Error); ok {
		return e.Code
	}
	return 0
}

func (r *Resource) Create(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		modelValue   any
		runtimeScope *RuntimeScope
	)
	modelValue = reflect.New(r.model.ModelType()).Interface()
	if err = json.NewDecoder(req.Body).Decode(modelValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioCreate
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if _, err = r.model.Create(ctx, modelValue); err != nil {
		r.Respond(res, req, err)
		return
	}
	r.Respond(res, req, CreateResult{
		ID: r.model.GetFieldValue(reflect.ValueOf(modelValue), r.model.primaryKey),
	})
}

func (r *Resource) Update(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		buf          []byte
		primaryKey   string
		runtimeScope *RuntimeScope
	)
	modelValue := reflect.New(r.model.ModelType()).Interface()
	mapValue := make(map[string]any)
	if buf, err = io.ReadAll(req.Body); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if err = json.Unmarshal(buf, modelValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if err = json.Unmarshal(buf, &mapValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioUpdate
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	primaryKey = r.findPrimaryKey(req, schema.ScenarioUpdate)
	columns := make([]string, 0, len(mapValue))
	for k := range mapValue {
		columns = append(columns, k)
	}
	var primaryKeyValue any
	if primaryKeyValue, err = r.model.Update(ctx, primaryKey, modelValue, columns...); err != nil {
		r.Respond(res, req, err)
		return
	}
	r.Respond(res, req, UpdateResult{
		ID: primaryKeyValue,
	})
}

func (r *Resource) Delete(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		runtimeScope *RuntimeScope
	)
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioDelete
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	primaryKey := r.findPrimaryKey(req, schema.ScenarioDelete)
	var primaryKeyValue any
	if primaryKeyValue, err = r.model.Delete(ctx, primaryKey); err != nil {
		r.Respond(res, req, err)
		return
	}
	r.Respond(res, req, DeletedResult{
		ID: primaryKeyValue,
	})
}

func (r *Resource) Detail(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		schemas      []schema.Schema
		modelValue   any
		valueFormat  string
		runtimeScope *RuntimeScope
		scenario     string
	)
	// 允许通过 ?scenario= 覆盖默认渲染场景；非法或缺省时回退到 detail。
	scenario = req.URL.Query().Get(QueryParamScenario)
	if !schema.IsValidScenario(scenario) {
		scenario = schema.ScenarioDetail
	}
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = scenario
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if schemas, err = schema.GetVisibleSchemas(ctx, r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, scenario); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Schemas = schemas
	valueFormat = req.URL.Query().Get(QueryParamFormat)
	if modelValue, err = r.model.Detail(ctx, r.findPrimaryKey(req, scenario)); err != nil {
		// 框架未命中的"未找到"统一映射成 ErrRecordNotFound（404）；
		// 其它 DB/业务错误按原样上抛，由 Respond 映射状态码。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrRecordNotFound
		}
		r.Respond(res, req, err)
		return
	}
	if r.formatter != nil {
		r.Respond(res, req, r.formatter.FormatModel(ctx, reflect.ValueOf(modelValue), schemas, r.model.GetDB().Statement, valueFormat))
	} else {
		r.Respond(res, req, modelValue)
	}
}

func (r *Resource) Search(res http.ResponseWriter, req *http.Request) {
	var (
		err           error
		pageIndex     int
		pageSize      int
		modelValues   any
		totalCount    int64
		valueFormat   string
		runtimeScope  *RuntimeScope
		searchSchemas []schema.Schema
		listSchemas   []schema.Schema
	)
	pageIndex, _ = strconv.Atoi(req.URL.Query().Get(QueryParamPage))
	pageSize, _ = strconv.Atoi(req.URL.Query().Get(QueryParamPageSize))
	valueFormat = req.URL.Query().Get(QueryParamFormat)
	if pageSize <= 0 {
		pageSize = 15
	}
	if pageIndex > 0 {
		pageIndex--
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioSearch
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if searchSchemas, err = schema.GetVisibleSchemas(ctx, r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioSearch); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	if listSchemas, err = schema.GetVisibleSchemas(ctx, r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioList); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Schemas = listSchemas
	queryBuilder := r.buildQuery(req, searchSchemas)
	if runtimeScope.TenantID != "" {
		queryBuilder.Where(TenantId, query.OpEq, runtimeScope.TenantID)
	}
	if totalCount, modelValues, err = r.model.Paginate(ctx, pageIndex, pageSize, queryBuilder); err != nil {
		r.Respond(res, req, err)
		return
	}
	result := &PageResult{
		Page:       pageIndex,
		PageSize:   pageSize,
		TotalCount: totalCount,
	}
	if r.formatter != nil {
		result.Data = r.formatter.FormatModels(ctx, modelValues, listSchemas, r.model.GetDB().Statement, valueFormat)
	} else {
		result.Data = modelValues
	}
	r.Respond(res, req, result)
}

func (r *Resource) Export(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		modelValues  any
		schemas      []schema.Schema
		runtimeScope *RuntimeScope
	)
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioExport
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if schemas, err = schema.GetVisibleSchemas(ctx, r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioExport); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	queryBuilder := r.buildQuery(req, schemas)
	if runtimeScope.TenantID != "" {
		queryBuilder.Where(TenantId, query.OpEq, runtimeScope.TenantID)
	}
	if modelValues, err = r.model.List(ctx, 0, 1000, queryBuilder); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	if r.formatter == nil {
		// 正常情况下 Register() 不会把 Export 路由挂载在没有 formatter 的 Resource 上；
		// 这里做兜底，防止误调。
		r.Respond(res, req, ErrUnavailable)
		return
	}
	res.Header().Set("Content-Type", "text/csv")
	res.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	res.Header().Set("Content-Disposition", fmt.Sprintf(
		"attachment;filename=%s_%s.csv",
		r.model.GetNaming().Singular,
		time.Now().Format(time.DateTime),
	))
	value := r.formatter.FormatModels(ctx, modelValues, schemas, r.model.GetDB().Statement, formats.FormatRaw)
	writer := csv.NewWriter(res)
	rows := make([]string, len(schemas))
	for i, field := range schemas {
		rows[i] = field.Label
	}
	_ = writer.Write(rows)
	if values, ok := value.([]any); ok {
		for _, val := range values {
			row, ok2 := val.(map[string]any)
			if !ok2 {
				continue
			}
			for i, field := range schemas {
				if v, ok := row[field.Column]; ok {
					rows[i] = fmt.Sprint(v)
				} else {
					rows[i] = ""
				}
			}
			_ = writer.Write(rows)
		}
	}
	writer.Flush()
}

func (r *Resource) OpenApi(res http.ResponseWriter, req *http.Request) {
	spec, err := openapi.NewGenerator().Generate(
		req.Context(),
		r.model.GetDB(),
		openapi.Config{
			Title:      r.model.GetNaming().ModuleName + " API",
			Version:    "1.0.0",
			ModuleName: r.model.GetNaming().ModuleName,
			TableName:  r.model.GetNaming().TableName,
			Singular:   r.model.GetNaming().Singular,
			Plural:     r.model.GetNaming().Pluralize,
			Prefix:     r.prefix,
			PrimaryKey: r.model.GetPrimaryKey(),
			Scenarios: []string{
				schema.ScenarioCreate,
				schema.ScenarioUpdate,
				schema.ScenarioDelete,
				schema.ScenarioDetail,
				schema.ScenarioSearch,
				schema.ScenarioExport,
			},
			BuildUri: r.buildUri,
			Model:    reflect.New(r.model.ModelType()).Interface(),
		},
	)
	res.Header().Set("Content-Type", "application/json")
	if err != nil {
		// OpenAPI 端点始终返回 JSON；错误体同样序列化为 JSON 对象。
		res.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(res).Encode(Error{
			Code:   0,
			Reason: err.Error(),
		})
		return
	}
	_ = json.NewEncoder(res).Encode(spec)
}

func (r *Resource) ModelValue() *Model {
	return r.model
}

func NewResource(model *Model, cfg ResourceConfig) *Resource {
	return &Resource{
		model:         model,
		router:        cfg.Router,
		responder:     cfg.Responder,
		formatter:     cfg.Formatter,
		prefix:        cfg.Prefix,
		tenantResolve: cfg.TenantResolve,
		userResolve:   cfg.UserResolve,
	}
}

// NewResourceWithOptions 根据任意模型实例(或 reflect.Type)一步创建并注册 Resource
func NewResourceWithOptions(model any, cfg ResourceConfig, opts ...Option) (resource *Resource, err error) {
	var modelValue *Model
	modelValue, err = NewModel(model, opts...)
	if err != nil {
		return
	}
	resource = NewResource(modelValue, cfg)
	resource.Register()
	return
}
