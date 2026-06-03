package openapi

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"git.nobla.cn/golang/rest/schema"
	"gorm.io/gorm"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

type Config struct {
	Title      string
	Version    string
	ModuleName string
	TableName  string
	Singular   string
	Plural     string
	Prefix     string
	PrimaryKey string
	Scenarios  []string
	BuildUri   func(scenario string) (method, uri string)
	Model      any // model type used to look up JSON tag for each field
}

func (g *Generator) Generate(ctx context.Context, db *gorm.DB, cfg Config) (*Spec, error) {
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   cfg.Title,
			Version: cfg.Version,
		},
		Paths:      make(map[string]PathItem),
		Components: Components{Schemas: make(map[string]*SchemaRef)},
	}

	state := &genState{
		db:      db,
		ctx:     ctx,
		spec:    spec,
		cfg:     cfg,
		visited: make(map[string]bool),
		stmts:   make(map[string]*gorm.Statement),
	}

	for _, scenario := range cfg.Scenarios {
		method, uri := cfg.BuildUri(scenario)
		if uri == "" {
			continue
		}

		openAPIPath := uriToOpenAPIPath(uri)
		visibleSchemas, err := schema.GetVisibleSchemas(ctx, db, cfg.ModuleName, cfg.TableName, scenario)
		if err != nil {
			continue
		}

		op := state.buildOperation(scenario, visibleSchemas)
		if op == nil {
			continue
		}

		item := spec.Paths[openAPIPath]
		switch strings.ToUpper(method) {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		case "PUT":
			item.Put = op
		case "DELETE":
			item.Delete = op
		}
		spec.Paths[openAPIPath] = item
	}

	return spec, nil
}

type genState struct {
	db      *gorm.DB
	ctx     context.Context
	spec    *Spec
	cfg     Config
	visited map[string]bool
	stmts   map[string]*gorm.Statement // key = module:table
	stmtMu  sync.Mutex
}

func (s *genState) buildOperation(scenario string, schemas []schema.Schema) *Operation {
	var (
		op          = &Operation{}
		parameters  []Parameter
		requestBody *RequestBody
		responses   = make(map[string]Response)
	)

	cfg := s.cfg
	baseName := schemaName(cfg.ModuleName, cfg.TableName)

	switch scenario {
	case schema.ScenarioCreate:
		op.Summary = fmt.Sprintf("创建 %s", cfg.Singular)
		op.OperationID = fmt.Sprintf("create%s", toPascal(cfg.Singular))
		requestBody = s.buildRequestBody(baseName+"Create", cfg.ModuleName, cfg.TableName, scenario)
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
					"id": {Type: "string"},
				}}},
			},
		}
	case schema.ScenarioUpdate:
		op.Summary = fmt.Sprintf("更新 %s", cfg.Singular)
		op.OperationID = fmt.Sprintf("update%s", toPascal(cfg.Singular))
		parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
		requestBody = s.buildRequestBody(baseName+"Update", cfg.ModuleName, cfg.TableName, scenario)
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
					"id": {Type: "string"},
				}}},
			},
		}
	case schema.ScenarioDelete:
		op.Summary = fmt.Sprintf("删除 %s", cfg.Singular)
		op.OperationID = fmt.Sprintf("delete%s", toPascal(cfg.Singular))
		parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"application/json": {Schema: &SchemaRef{Type: "object", Properties: map[string]*SchemaRef{
					"id": {Type: "string"},
				}}},
			},
		}
	case schema.ScenarioDetail:
		op.Summary = fmt.Sprintf("获取 %s 详情", cfg.Singular)
		op.OperationID = fmt.Sprintf("detail%s", toPascal(cfg.Singular))
		parameters = append(parameters, s.buildIDParameter(cfg.PrimaryKey, schemas))
		parameters = append(parameters, Parameter{
			Name:        "format",
			In:          "query",
			Description: "返回格式",
			Schema:      &SchemaRef{Type: "string", Enum: []any{"raw", "both"}},
		})
		ref := s.buildSchemaRef(cfg.ModuleName, cfg.TableName, scenario)
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"application/json": {Schema: ref},
			},
		}
	case schema.ScenarioSearch:
		op.Summary = fmt.Sprintf("查询 %s 列表", cfg.Plural)
		op.OperationID = fmt.Sprintf("list%s", toPascal(cfg.Plural))
		parameters = append(parameters,
			Parameter{Name: "page", In: "query", Schema: &SchemaRef{Type: "integer"}},
			Parameter{Name: "page_size", In: "query", Schema: &SchemaRef{Type: "integer"}},
			Parameter{Name: "query", In: "query", Schema: &SchemaRef{Type: "string"}, Description: "AST 查询表达式"},
			Parameter{Name: "format", In: "query", Schema: &SchemaRef{Type: "string", Enum: []any{"raw", "both"}}},
		)
		itemRef := s.buildSchemaRef(cfg.ModuleName, cfg.TableName, schema.ScenarioList)
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &SchemaRef{
						Type: "object",
						Properties: map[string]*SchemaRef{
							"page":        {Type: "integer"},
							"page_size":   {Type: "integer"},
							"total_count": {Type: "integer"},
							"data":        {Type: "array", Items: itemRef},
						},
					},
				},
			},
		}
	case schema.ScenarioExport:
		op.Summary = fmt.Sprintf("导出 %s", cfg.Plural)
		op.OperationID = fmt.Sprintf("export%s", toPascal(cfg.Plural))
		parameters = append(parameters,
			Parameter{Name: "query", In: "query", Schema: &SchemaRef{Type: "string"}, Description: "AST 查询表达式"},
		)
		responses["200"] = Response{
			Description: "成功",
			Content: map[string]MediaType{
				"text/csv": {Schema: &SchemaRef{Type: "string", Format: "binary"}},
			},
		}
	default:
		return nil
	}

	op.Parameters = parameters
	op.RequestBody = requestBody
	op.Responses = responses
	return op
}

func (s *genState) buildIDParameter(primaryKey string, schemas []schema.Schema) Parameter {
	var pkType string
	for _, sc := range schemas {
		if sc.Column == primaryKey && sc.PrimaryKey > 0 {
			pkType = mapSchemaType(sc.Type)
			break
		}
	}
	if pkType == "" {
		pkType = "string"
	}
	return Parameter{
		Name:     "id",
		In:       "path",
		Required: true,
		Schema:   &SchemaRef{Type: pkType},
	}
}

func (s *genState) buildRequestBody(refName, module, table, scenario string) *RequestBody {
	ref := s.buildSchemaRef(module, table, scenario)
	if ref.Ref == "" {
		// Schema generation failed; fall back to inline object instead of self-reference
		return &RequestBody{
			Content: map[string]MediaType{
				"application/json": {Schema: &SchemaRef{Type: "object"}},
			},
		}
	}
	return &RequestBody{
		Content: map[string]MediaType{
			"application/json": {Schema: &SchemaRef{Ref: "#/components/schemas/" + refName}},
		},
	}
}

func (s *genState) buildSchemaRef(module, table, scenario string) *SchemaRef {
	name := schemaName(module, table)

	// For the primary model, use scenario-specific naming for Create/Update
	if module == s.cfg.ModuleName && table == s.cfg.TableName {
		switch scenario {
		case schema.ScenarioCreate:
			name = name + "Create"
		case schema.ScenarioUpdate:
			name = name + "Update"
		default:
			// Detail, List, Export, Search use base name
		}
	}

	if s.visited[name] {
		return &SchemaRef{Ref: "#/components/schemas/" + name}
	}
	s.visited[name] = true

	schemas, err := schema.GetVisibleSchemas(s.ctx, s.db, module, table, scenario)
	if err != nil || len(schemas) == 0 {
		if err != nil {
			log.Printf("[openapi] GetVisibleSchemas failed for %s.%s (scenario=%s): %v", module, table, scenario, err)
		}
		return &SchemaRef{Type: "object"}
	}

	ref := &SchemaRef{
		Type:       "object",
		Properties: make(map[string]*SchemaRef),
	}

	for _, sc := range schemas {
		prop := s.schemaToProperty(sc)
		// Property key MUST use JSON tag (the API contract), not GORM DBName.
		// For relations, sc.Column is the struct field name; the JSON tag is
		// the API name.
		key := s.jsonName(module, table, sc.Column)
		ref.Properties[key] = prop
		for _, req := range sc.Rules.Required {
			if req == scenario {
				ref.Required = append(ref.Required, key)
				break
			}
		}
	}

	s.spec.Components.Schemas[name] = ref
	return &SchemaRef{Ref: "#/components/schemas/" + name}
}

func (s *genState) schemaToProperty(sc schema.Schema) *SchemaRef {
	if sc.Relations.Type != "" {
		assocRef := s.buildSchemaRef(sc.Relations.Module, sc.Relations.Table, schema.ScenarioDetail)
		switch sc.Relations.Type {
		case "has_many", "many_to_many":
			// Represent association as array of referenced schema
			return &SchemaRef{Type: "array", Items: assocRef}
		default:
			return assocRef
		}
	}

	prop := &SchemaRef{
		Type:        mapSchemaType(sc.Type),
		Format:      mapSchemaFormat(sc.Type),
		Description: sc.Label,
	}
	if sc.PrimaryKey > 0 {
		prop.ReadOnly = true
	}
	return prop
}

func mapSchemaType(t string) string {
	switch t {
	case schema.TypeInteger:
		return "integer"
	case schema.TypeFloat:
		return "number"
	case schema.TypeBoolean:
		return "boolean"
	default:
		return "string"
	}
}

func mapSchemaFormat(t string) string {
	switch t {
	case schema.FormatDate:
		return "date"
	case schema.FormatDatetime, schema.FormatTimestamp:
		return "date-time"
	default:
		return ""
	}
}

func uriToOpenAPIPath(uri string) string {
	parts := strings.Split(uri, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// stmtFor returns (and caches) a parsed gorm.Statement for the model in cfg.Model
// whose DB table matches the given tableName. If cfg.Model is nil or the table
// doesn't match, returns nil.
func (s *genState) stmtFor(module, table string) *gorm.Statement {
	key := module + ":" + table
	s.stmtMu.Lock()
	defer s.stmtMu.Unlock()
	if st, ok := s.stmts[key]; ok {
		return st
	}
	if s.cfg.Model == nil {
		return nil
	}
	modelType := reflect.TypeOf(s.cfg.Model)
	if modelType == nil {
		return nil
	}
	// unwrap pointer
	for modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	stmt := &gorm.Statement{DB: s.db, Table: table}
	if err := stmt.Parse(reflect.New(modelType).Interface()); err != nil {
		log.Printf("[openapi] parse model %v failed: %v", s.cfg.Model, err)
		return nil
	}
	if stmt.Table != table {
		return nil
	}
	s.stmts[key] = stmt
	return stmt
}

// jsonName returns the JSON tag name of the field whose GORM DBName matches
// column. Falls back to column when the field is not found.
func (s *genState) jsonName(module, table, column string) string {
	stmt := s.stmtFor(module, table)
	if stmt == nil || stmt.Schema == nil {
		return column
	}
	field := stmt.Schema.LookUpField(column)
	if field == nil {
		// also try matching struct field name
		for _, f := range stmt.Schema.Fields {
			if f.Name == column {
				field = f
				break
			}
		}
	}
	if field == nil {
		return column
	}
	tag := field.StructField.Tag.Get("json")
	if tag == "" || tag == "-" {
		return column
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return column
	}
	return parts[0]
}

func schemaName(module, table string) string {
	return toPascal(module) + toPascal(table)
}

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	var result string
	for _, p := range parts {
		if p == "" {
			continue
		}
		result += strings.ToUpper(p[:1]) + p[1:]
	}
	return result
}
