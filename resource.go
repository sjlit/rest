package rest

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"git.nobla.cn/golang/rest/formats"
	"git.nobla.cn/golang/rest/openapi"
	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
)

type (
	ResourceOption[T any] func(*Resource[T])

	Resource[T any] struct {
		model         *Model[T]
		prefix        string
		router        Router
		responder     Responder
		formatter     *formats.Formatter
		tenantResolve ResolveTenantFunc
		userResolve   ResolveUserFunc
		openAPISpec   []byte
	}
)

func WithRouter[T any](router Router) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.router = router
	}
}

func WithResponder[T any](responder Responder) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.responder = responder
	}
}

func WithFormatter[T any](formatter *formats.Formatter) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.formatter = formatter
	}
}

func WithPrefix[T any](prefix string) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.prefix = prefix
	}
}

func WithTenantResolve[T any](tenantResolve ResolveTenantFunc) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.tenantResolve = tenantResolve
	}
}

func WithUserResolve[T any](userResolve ResolveUserFunc) ResourceOption[T] {
	return func(r *Resource[T]) {
		r.userResolve = userResolve
	}
}

func (r *Resource[T]) buildUri(scenario string) (method string, uri string) {
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
	}
	return
}

func (r *Resource[T]) buildQuery(req *http.Request, schemas []schema.Schema) *query.Builder {
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
			if ss := strings.Split(formValue, sep); len(ss) == 2 {
				builder.WhereGroup(func(b *query.Builder) {
					b.Where(row.Column, query.OpGte, ss[0])
					b.Where(row.Column, query.OpLte, ss[1])
				})
			} else {
				builder.Where(row.Column, query.OpEq, formValue)
			}
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

func (r *Resource[T]) findPrimaryKey(req *http.Request, scenario string) string {
	_, fullUri := r.buildUri(scenario)
	reqPath := req.URL.Path
	fullParts := strings.Split(fullUri, "/")
	reqParts := strings.Split(reqPath, "/")
	for i, part := range fullParts {
		if strings.HasPrefix(part, ":") && i < len(reqParts) {
			return reqParts[i]
		}
	}
	return ""
}

func (r *Resource[T]) getRuntimeScope(req *http.Request) (runtimeScope *RuntimeScope, err error) {
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

func (r *Resource[T]) Register() {
	var (
		method string
		uri    string
	)
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
	if r.model.HasScenario(schema.ScenarioExport) {
		method, uri = r.buildUri(schema.ScenarioExport)
		r.router.Handle(method, uri, r.Export)
	}
	if r.model.OpenAPIEnabled() {
		spec, err := openapi.NewGenerator().Generate(
			context.Background(),
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
			},
		)
		if err != nil {
			log.Printf("[OpenAPI] failed to generate spec for %s: %v", r.model.GetNaming().ModuleName, err)
		} else {
			bytes, err := json.Marshal(spec)
			if err != nil {
				log.Printf("[OpenAPI] failed to marshal spec for %s: %v", r.model.GetNaming().ModuleName, err)
			} else {
				r.openAPISpec = bytes
				openAPIPath := path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, "openapi.json")
				r.router.Handle(http.MethodGet, openAPIPath, r.ServeOpenAPI)
			}
		}
	}
}

func (r *Resource[T]) Respond(res http.ResponseWriter, req *http.Request, data any) {
	if r.responder != nil {
		r.responder.Respond(res, req, data)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err, ok := data.(error); ok {
		res.WriteHeader(http.StatusServiceUnavailable)
		res.Write([]byte(err.Error()))
		return
	}
	json.NewEncoder(res).Encode(data)
}

func (r *Resource[T]) ServeOpenAPI(res http.ResponseWriter, req *http.Request) {
	if len(r.openAPISpec) == 0 {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(r.openAPISpec)
}

func (r *Resource[T]) Create(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		modelValue   T
		runtimeScope *RuntimeScope
	)
	if err = json.NewDecoder(req.Body).Decode(&modelValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioCreate
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if _, err = r.model.Create(ctx, &modelValue); err != nil {
		r.Respond(res, req, err)
		return
	}
	r.Respond(res, req, CreateResult{
		ID: r.model.GetFieldValue(reflect.ValueOf(modelValue), r.model.primaryKey),
	})
}

func (r *Resource[T]) Update(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		buf          []byte
		primaryKey   string
		runtimeScope *RuntimeScope
	)
	modelValue := new(T)
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
		r.Respond(res, req, ErrUpdateFailed)
	} else {
		r.Respond(res, req, UpdateResult{
			ID: primaryKeyValue,
		})
	}
}

func (r *Resource[T]) Delete(res http.ResponseWriter, req *http.Request) {
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
		r.Respond(res, req, ErrDeleteFailed)
	} else {
		r.Respond(res, req, DeletedResult{
			ID: primaryKeyValue,
		})
	}
}

func (r *Resource[T]) Detail(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		schemas      []schema.Schema
		modelValue   *T
		valueFormat  string
		runtimeScope *RuntimeScope
	)
	if runtimeScope, err = r.getRuntimeScope(req); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Scenario = schema.ScenarioDetail
	ctx := WithRuntimeScope(req.Context(), runtimeScope)
	if schemas, err = schema.GetVisibleSchemas(ctx, r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioDetail); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	runtimeScope.Schemas = schemas
	valueFormat = req.URL.Query().Get(QueryParamFormat)
	if modelValue, err = r.model.Detail(ctx, r.findPrimaryKey(req, schema.ScenarioDetail)); err != nil {
		r.Respond(res, req, ErrRecordNotFound)
		return
	}
	if r.formatter != nil {
		r.Respond(res, req, r.formatter.FormatModel(ctx, reflect.ValueOf(modelValue), schemas, r.model.GetDB().Statement, valueFormat))
	} else {
		r.Respond(res, req, modelValue)
	}
}

func (r *Resource[T]) Search(res http.ResponseWriter, req *http.Request) {
	var (
		err           error
		pageIndex     int
		pageSize      int
		modelValues   []*T
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
		r.Respond(res, req, ErrUnavailable)
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

func (r *Resource[T]) Export(res http.ResponseWriter, req *http.Request) {
	var (
		err          error
		modelValues  []*T
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

func NewResource[T any](model *Model[T], opts ...ResourceOption[T]) *Resource[T] {
	r := &Resource[T]{
		model: model,
	}
	for _, cb := range opts {
		cb(r)
	}
	return r
}
