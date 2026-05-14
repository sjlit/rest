package openapi

import (
	"testing"

	"git.nobla.cn/golang/rest/schema"
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
