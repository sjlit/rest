package openapi

import (
    "testing"

    "git.nobla.cn/golang/rest/schema"
)

func TestMapSchemaType(t *testing.T) {
    tests := []struct {
        input    string
        wantType string
        wantFmt  string
    }{
        {schema.TypeInteger, "integer", ""},
        {schema.TypeFloat, "number", ""},
        {schema.TypeBoolean, "boolean", ""},
        {schema.TypeString, "string", ""},
        {schema.FormatDate, "string", "date"},
        {schema.FormatDatetime, "string", "date-time"},
        {schema.FormatTimestamp, "string", "date-time"},
    }
    for _, tt := range tests {
        if got := mapSchemaType(tt.input); got != tt.wantType {
            t.Errorf("mapSchemaType(%q) = %q, want %q", tt.input, got, tt.wantType)
        }
        if got := mapSchemaFormat(tt.input); got != tt.wantFmt {
            t.Errorf("mapSchemaFormat(%q) = %q, want %q", tt.input, got, tt.wantFmt)
        }
    }
}

func TestUriToOpenAPIPath(t *testing.T) {
    tests := []struct {
        uri  string
        want string
    }{
        {"/api/v1/rest/user/:id", "/api/v1/rest/user/{id}"},
        {"/api/v1/rest/users", "/api/v1/rest/users"},
        {"/api/v1/rest/user/detail/:id", "/api/v1/rest/user/detail/{id}"},
    }
    for _, tt := range tests {
        if got := uriToOpenAPIPath(tt.uri); got != tt.want {
            t.Errorf("uriToOpenAPIPath(%q) = %q, want %q", tt.uri, got, tt.want)
        }
    }
}

func TestToPascal(t *testing.T) {
    tests := []struct {
        in   string
        want string
    }{
        {"rest", "Rest"},
        {"integ_users", "IntegUsers"},
        {"", ""},
    }
    for _, tt := range tests {
        if got := toPascal(tt.in); got != tt.want {
            t.Errorf("toPascal(%q) = %q, want %q", tt.in, got, tt.want)
        }
    }
}
