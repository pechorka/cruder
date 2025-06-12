package swaggergen

import (
	"reflect"
	"strings"
)

// OpenAPI represents the root OpenAPI 3.0 specification
type OpenAPI struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers,omitempty"`
	Paths      map[string]PathItem `json:"paths"`
	Components *Components         `json:"components,omitempty"`
}

// Info provides metadata about the API
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Server represents a server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations available on a single path
type PathItem struct {
	GET    *Operation `json:"get,omitempty"`
	POST   *Operation `json:"post,omitempty"`
	PUT    *Operation `json:"put,omitempty"`
	DELETE *Operation `json:"delete,omitempty"`
	PATCH  *Operation `json:"patch,omitempty"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags        []string            `json:"tags,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

// Parameter describes a single operation parameter
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// Response describes a single response from an API Operation
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType provides schema and examples for the media type
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Components holds a set of reusable objects for different aspects of the OAS
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// Schema represents a JSON Schema
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Description          string             `json:"description,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty"`
}

// HandlerInfo contains information about a registered handler
type HandlerInfo struct {
	Name         string
	Path         string
	Method       string
	RequestType  reflect.Type
	ResponseType reflect.Type
	Tags         []string
	Summary      string
	Description  string
}

// Generator generates OpenAPI specifications
type Generator struct {
	openapi    *OpenAPI
	components *Components
	schemas    map[string]*Schema
}

// NewGenerator creates a new swagger generator
func NewGenerator() *Generator {
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	return &Generator{
		openapi: &OpenAPI{
			OpenAPI: "3.0.0",
			Info: Info{
				Title:   "API Documentation",
				Version: "1.0.0",
			},
			Paths:      make(map[string]PathItem),
			Components: components,
		},
		components: components,
		schemas:    make(map[string]*Schema),
	}
}

// SetInfo sets the API info
func (g *Generator) SetInfo(title, description, version string) {
	g.openapi.Info.Title = title
	g.openapi.Info.Description = description
	g.openapi.Info.Version = version
}

// AddServer adds a server to the OpenAPI spec
func (g *Generator) AddServer(url, description string) {
	g.openapi.Servers = append(g.openapi.Servers, Server{
		URL:         url,
		Description: description,
	})
}

// RegisterHandler registers a handler for swagger generation
func (g *Generator) RegisterHandler(info HandlerInfo) {
	pathItem := g.openapi.Paths[info.Path]

	operation := &Operation{
		Tags:        info.Tags,
		Summary:     info.Summary,
		Description: info.Description,
		OperationID: info.Name,
		Responses:   make(map[string]Response),
	}

	// Add request body if request type is not empty
	if info.RequestType != nil && info.RequestType.Kind() != reflect.Invalid {
		reqSchema := g.generateSchema(info.RequestType)
		operation.RequestBody = &RequestBody{
			Description: "Request body",
			Content: map[string]MediaType{
				"application/json": {
					Schema: reqSchema,
				},
			},
			Required: true,
		}
	}

	// Add response
	if info.ResponseType != nil && info.ResponseType.Kind() != reflect.Invalid {
		respSchema := g.generateSchema(info.ResponseType)
		operation.Responses["200"] = Response{
			Description: "Successful response",
			Content: map[string]MediaType{
				"application/json": {
					Schema: respSchema,
				},
			},
		}
	} else {
		operation.Responses["200"] = Response{
			Description: "Successful response",
		}
	}

	// Add error response
	operation.Responses["500"] = Response{
		Description: "Internal server error",
	}

	// Set operation based on method
	switch strings.ToUpper(info.Method) {
	case "GET":
		pathItem.GET = operation
	case "POST":
		pathItem.POST = operation
	case "PUT":
		pathItem.PUT = operation
	case "DELETE":
		pathItem.DELETE = operation
	case "PATCH":
		pathItem.PATCH = operation
	}

	g.openapi.Paths[info.Path] = pathItem
}

// generateSchema generates a JSON schema for a Go type
func (g *Generator) generateSchema(t reflect.Type) *Schema {
	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeName := g.getTypeName(t)

	// Check if schema already exists
	if typeName != "" {
		if _, exists := g.schemas[typeName]; exists {
			return &Schema{Ref: "#/components/schemas/" + typeName}
		}
	}

	schema := &Schema{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		itemSchema := g.generateSchema(t.Elem())
		schema.Items = itemSchema
	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = true
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]*Schema)
		var required []string

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			fieldName := field.Name

			// Parse json tag
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					fieldName = parts[0]
				}

				// Skip if json:"-"
				if jsonTag == "-" {
					continue
				}

				// Check if field is required (not omitempty)
				isRequired := true
				for _, part := range parts[1:] {
					if part == "omitempty" {
						isRequired = false
						break
					}
				}

				if isRequired {
					required = append(required, fieldName)
				}
			} else {
				// Default behavior: field is required if not a pointer
				if field.Type.Kind() != reflect.Ptr {
					required = append(required, fieldName)
				}
			}

			fieldSchema := g.generateSchema(field.Type)
			schema.Properties[fieldName] = fieldSchema
		}

		if len(required) > 0 {
			schema.Required = required
		}

		// Store schema in components if it's a named type
		if typeName != "" {
			g.schemas[typeName] = schema
			g.components.Schemas[typeName] = schema
			return &Schema{Ref: "#/components/schemas/" + typeName}
		}
	}

	return schema
}

// getTypeName returns a clean type name for schema references
func (g *Generator) getTypeName(t reflect.Type) string {
	if t.Name() != "" {
		return t.Name()
	}

	// For anonymous types, create a name based on the structure
	switch t.Kind() {
	case reflect.Slice:
		return "ArrayOf" + g.getTypeName(t.Elem())
	case reflect.Map:
		return "MapOf" + g.getTypeName(t.Elem())
	case reflect.Ptr:
		return g.getTypeName(t.Elem())
	}

	return ""
}

// GenerateJSON generates the OpenAPI specification as JSON
func (g *Generator) Schema() *OpenAPI {
	return g.openapi
}
