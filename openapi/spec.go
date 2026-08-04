package openapi

type Spec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string     `json:"name"`
	In          string     `json:"in"`
	Required    bool       `json:"required,omitempty"`
	Schema      *SchemaRef `json:"schema"`
	Description string     `json:"description,omitempty"`
}

type RequestBody struct {
	Content map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema *SchemaRef `json:"schema"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type SchemaRef struct {
	Ref         string                `json:"$ref,omitempty"`
	Type        string                `json:"type,omitempty"`
	Format      string                `json:"format,omitempty"`
	Items       *SchemaRef            `json:"items,omitempty"`
	Properties  map[string]*SchemaRef `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
	ReadOnly    bool                  `json:"readOnly,omitempty"`
	Enum        []any                 `json:"enum,omitempty"`
	Description string                `json:"description,omitempty"`
}

type Components struct {
	Schemas map[string]*SchemaRef `json:"schemas"`
}
