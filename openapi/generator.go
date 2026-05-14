package openapi

import (
	"context"
	"strings"

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
	// TODO: Task 4
	_ = ctx
	_ = db
	_ = cfg
	return spec, nil
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
