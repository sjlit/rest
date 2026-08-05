package openapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sjlit/rest/v3/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMapSchemaType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{schema.TypeInteger, "integer"},
		{schema.TypeFloat, "number"},
		{schema.TypeBoolean, "boolean"},
		{schema.TypeString, "string"},
		{schema.FormatDate, "string"},
		{schema.FormatDatetime, "string"},
		{schema.FormatTimestamp, "string"},
		{"unknown", "string"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := mapSchemaType(tt.input); got != tt.want {
				t.Errorf("mapSchemaType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapSchemaFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{schema.TypeInteger, ""},
		{schema.TypeFloat, ""},
		{schema.TypeBoolean, ""},
		{schema.TypeString, ""},
		{schema.FormatDate, "date"},
		{schema.FormatDatetime, "date-time"},
		{schema.FormatTimestamp, "date-time"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := mapSchemaFormat(tt.input); got != tt.want {
				t.Errorf("mapSchemaFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUriToOpenAPIPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		uri  string
		want string
	}{
		{"/api/v1/rest/user/:id", "/api/v1/rest/user/{id}"},
		{"/api/v1/rest/users", "/api/v1/rest/users"},
		{"/api/v1/rest/user/detail/:id", "/api/v1/rest/user/detail/{id}"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			t.Parallel()
			if got := uriToOpenAPIPath(tt.uri); got != tt.want {
				t.Errorf("uriToOpenAPIPath(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestToPascal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"rest", "Rest"},
		{"integ_users", "IntegUsers"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := toPascal(tt.input); got != tt.want {
				t.Errorf("toPascal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSchemaName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		module string
		table  string
		want   string
	}{
		{"api", "user", "ApiUser"},
		{"", "user", "User"},
		{"api", "", "Api"},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.module+"_"+tt.table, func(t *testing.T) {
			t.Parallel()
			if got := schemaName(tt.module, tt.table); got != tt.want {
				t.Errorf("schemaName(%q, %q) = %q, want %q", tt.module, tt.table, got, tt.want)
			}
		})
	}
}

type TestOrder struct {
	ID     uint `json:"id" gorm:"primarykey"`
	UserID uint `json:"user_id"`
	Total  int  `json:"total"`
}

type TestUser struct {
	ID     uint        `json:"id" gorm:"primarykey"`
	Name   string      `json:"name" gorm:"size:100"`
	Age    int         `json:"age"`
	Orders []TestOrder `json:"orders" gorm:"foreignKey:UserID" relation:"has_many:Orders:openapi_test:test_orders"`
}

func TestGenerate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "cgo") || strings.Contains(err.Error(), "stub") {
			t.Skip("sqlite requires cgo")
		}
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&schema.Schema{}, &TestUser{}, &TestOrder{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := schema.AutoMigrate(context.Background(), db, &TestUser{}, "openapi_test"); err != nil {
		t.Fatalf("auto migrate schema: %v", err)
	}
	if _, err := schema.AutoMigrate(context.Background(), db, &TestOrder{}, "openapi_test"); err != nil {
		t.Fatalf("auto migrate schema: %v", err)
	}

	g := NewGenerator()
	cfg := Config{
		Title:      "Test API",
		Version:    "1.0.0",
		ModuleName: "openapi_test",
		TableName:  "test_users",
		Singular:   "user",
		Plural:     "users",
		Prefix:     "/api/v1",
		PrimaryKey: "id",
		Model:      (*TestUser)(nil),
		Scenarios: []string{
			schema.ScenarioCreate,
			schema.ScenarioUpdate,
			schema.ScenarioDelete,
			schema.ScenarioDetail,
			schema.ScenarioSearch,
			schema.ScenarioExport,
		},
		BuildUri: func(scenario string) (string, string) {
			switch scenario {
			case schema.ScenarioCreate:
				return "POST", "/api/v1/openapi_test/user"
			case schema.ScenarioUpdate:
				return "PUT", "/api/v1/openapi_test/user/:id"
			case schema.ScenarioDelete:
				return "DELETE", "/api/v1/openapi_test/user/:id"
			case schema.ScenarioDetail:
				return "GET", "/api/v1/openapi_test/user/detail/:id"
			case schema.ScenarioSearch:
				return "GET", "/api/v1/openapi_test/users"
			case schema.ScenarioExport:
				return "GET", "/api/v1/openapi_test/user/export"
			}
			return "", ""
		},
	}

	spec, err := g.Generate(context.Background(), db, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify paths exist
	if len(spec.Paths) != 5 {
		t.Fatalf("expected 5 paths, got %d", len(spec.Paths))
	}
	if spec.Paths["/api/v1/openapi_test/user"].Post == nil {
		t.Fatal("expected POST /user")
	}
	if spec.Paths["/api/v1/openapi_test/user/{id}"].Put == nil {
		t.Fatal("expected PUT /user/{id}")
	}
	if spec.Paths["/api/v1/openapi_test/user/{id}"].Delete == nil {
		t.Fatal("expected DELETE /user/{id}")
	}
	if spec.Paths["/api/v1/openapi_test/user/detail/{id}"].Get == nil {
		t.Fatal("expected GET /user/detail/{id}")
	}
	if spec.Paths["/api/v1/openapi_test/users"].Get == nil {
		t.Fatal("expected GET /users")
	}
	if spec.Paths["/api/v1/openapi_test/user/export"].Get == nil {
		t.Fatal("expected GET /user/export")
	}

	// Verify components/schemas exist
	if len(spec.Components.Schemas) == 0 {
		t.Fatal("expected non-empty schemas")
	}

	baseName := schemaName("openapi_test", "test_users")
	if _, ok := spec.Components.Schemas[baseName]; !ok {
		t.Fatalf("expected schema %q in components", baseName)
	}
	userSchema := spec.Components.Schemas[baseName]
	if userSchema == nil || userSchema.Properties == nil {
		t.Fatalf("expected %q to have properties", baseName)
	}
	// Verify property keys use JSON tag names (not GORM DBName)
	if _, ok := userSchema.Properties["id"]; !ok {
		t.Errorf("expected %q to have 'id' property (from json tag)", baseName)
	}
	if _, ok := userSchema.Properties["name"]; !ok {
		t.Errorf("expected %q to have 'name' property (from json tag)", baseName)
	}
	if _, ok := userSchema.Properties["age"]; !ok {
		t.Errorf("expected %q to have 'age' property (from json tag)", baseName)
	}
	// Orders field has json:"orders" tag, must be lowercased
	if _, ok := userSchema.Properties["Orders"]; ok {
		t.Errorf("unexpected 'Orders' property — should use JSON tag 'orders'")
	}
	if ordersProp, ok := userSchema.Properties["orders"]; !ok {
		t.Errorf("expected %q to have 'orders' property (from json tag)", baseName)
	} else if ordersProp.Type != "array" || ordersProp.Items == nil {
		t.Errorf("expected 'orders' to be an array with items")
	}

	// Verify association schema exists
	assocName := schemaName("openapi_test", "test_orders")
	if _, ok := spec.Components.Schemas[assocName]; !ok {
		t.Fatalf("expected association schema %q in components", assocName)
	}

	// Verify JSON serialization works
	_, err = json.Marshal(spec)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
}
