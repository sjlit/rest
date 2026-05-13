package rest

import (
	"encoding/json"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"

	"git.nobla.cn/golang/rest/formats"
	"git.nobla.cn/golang/rest/query"
	"git.nobla.cn/golang/rest/schema"
)

type (
	ResourceOption[T any] func(*Resource[T])

	Resource[T any] struct {
		model     *Model[T]
		prefix    string
		router    Router
		responder Responder
		formatter *formats.Formatter
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
		uri = path.Join(r.prefix, r.model.GetNaming().ModuleName, r.model.GetNaming().Singular, "export", ":id")
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
				builder.Where(row.Column, query.OpLike, formValue)
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
					builder.Where(row.Column, query.OpLike, formValue)
				}
			} else {
				builder.Where(row.Column, query.OpEq, formValue)
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
	// if r.model.HasScenario(schema.ScenarioExport) {
	// 	method, uri = r.buildUri(schema.ScenarioExport)
	// 	r.router.Handle(method, uri, r.Export)
	// }
}

func (r *Resource[T]) Respond(res http.ResponseWriter, req *http.Request, data any) {
	if r.responder != nil {
		r.responder.Respond(res, req, data)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(data)
}

func (r *Resource[T]) Create(res http.ResponseWriter, req *http.Request) {
	var (
		err        error
		modelValue T
	)
	if err = json.NewDecoder(req.Body).Decode(&modelValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	if _, err = r.model.Create(req.Context(), &modelValue); err != nil {
		r.Respond(res, req, ErrCreateFailed)
	} else {
		r.Respond(res, req, modelValue)
	}
}

func (r *Resource[T]) Update(res http.ResponseWriter, req *http.Request) {
	var (
		err        error
		modelValue T
		primaryKey string
	)
	if err = json.NewDecoder(req.Body).Decode(&modelValue); err != nil {
		r.Respond(res, req, ErrPayloadInvalid)
		return
	}
	primaryKey = r.findPrimaryKey(req, schema.ScenarioUpdate)
	if _, err = r.model.Update(req.Context(), primaryKey, modelValue); err != nil {
		r.Respond(res, req, ErrUpdateFailed)
	} else {
		r.Respond(res, req, UpdateResult{
			ID: primaryKey,
		})
	}
}

func (r *Resource[T]) Delete(res http.ResponseWriter, req *http.Request) {
	var (
		err error
	)
	primaryKeyValue := r.findPrimaryKey(req, schema.ScenarioDelete)
	if err = r.model.Delete(req.Context(), primaryKeyValue); err != nil {
		r.Respond(res, req, ErrDeleteFailed)
	} else {
		r.Respond(res, req, DeletedResult{
			ID: primaryKeyValue,
		})
	}
}

func (r *Resource[T]) Detail(res http.ResponseWriter, req *http.Request) {
	var (
		err        error
		schemas    []schema.Schema
		modelValue *T
	)
	if schemas, err = schema.GetVisibleSchemas(req.Context(), r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioDetail); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	if modelValue, err = r.model.Detail(req.Context(), r.findPrimaryKey(req, schema.ScenarioDetail)); err != nil {
		r.Respond(res, req, ErrRecordNotFound)
		return
	}
	if r.formatter != nil {
		r.Respond(res, req, r.formatter.FormatModel(req.Context(), reflect.ValueOf(modelValue), schemas, r.model.GetDB().Statement, ""))
	} else {
		r.Respond(res, req, modelValue)
	}
}

func (r *Resource[T]) Search(res http.ResponseWriter, req *http.Request) {
	var (
		err        error
		pageIndex  int
		pageSize   int
		schemas    []schema.Schema
		result     *PageResult[T]
	)
	pageIndex, _ = strconv.Atoi(req.URL.Query().Get("page"))
	pageSize, _ = strconv.Atoi(req.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 15
	}
	if pageIndex > 0 {
		pageIndex--
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	if schemas, err = schema.GetVisibleSchemas(req.Context(), r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioSearch); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	queryBuilder := r.buildQuery(req, schemas)
	if result, err = r.model.Paginate(req.Context(), pageIndex+1, pageSize, queryBuilder); err != nil {
		r.Respond(res, req, ErrUnavailable)
		return
	}
	if r.formatter != nil {
		result.Data = r.formatter.FormatModels(req.Context(), result.Data, schemas, r.model.GetDB().Statement, "").([]*T)
	}
	r.Respond(res, req, result)
}

// func (r *Resource[T]) Export(res http.ResponseWriter, req *http.Request) {
// 	var (
// 		err         error
// 		schemas     []schema.Schema
// 		modelValues []*T
// 	)
// 	if err = r.model.Delete(req.Context(), 0); err != nil {
// 		r.Respond(res, req, ErrCreateFailed)
// 	}
// 	if schemas, err = schema.GetVisibleSchemas(req.Context(), r.model.GetDB(), r.model.GetNaming().ModuleName, r.model.GetNaming().TableName, schema.ScenarioExport); err != nil {
// 		r.Respond(res, req, ErrUnavailable)
// 		return
// 	}
// 	if _, modelValues, err = r.model.Find(req.Context(), 0, 0, nil); err != nil {
// 		r.Respond(res, req, ErrRecordNotFound)
// 		return
// 	}
// 	for _, row := range modelValues {

// 	}
// }

func NewResource[T any](model *Model[T], opts ...ResourceOption[T]) *Resource[T] {
	r := &Resource[T]{
		model: model,
	}
	for _, cb := range opts {
		cb(r)
	}
	return r
}
