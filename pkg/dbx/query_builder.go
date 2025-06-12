package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// DB interface for executing queries
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// InsertBuilder represents an insert query builder
type InsertBuilder[T any] struct {
	table       string
	inputType   reflect.Type
	inputFields []fieldInfo
}

// InsertReturningBuilder represents an insert query builder with returning clause
type InsertReturningBuilder[T, R any] struct {
	insert          *InsertBuilder[T]
	returningType   reflect.Type
	returningFields []fieldInfo
}

// CompiledInsertQuery represents a compiled insert query
type CompiledInsertQuery[T, R any] struct {
	query           string
	inputFields     []fieldInfo
	returningFields []fieldInfo
	hasReturning    bool
}

// ExecutableQuery represents a query ready for execution
type ExecutableQuery[T, R any] struct {
	compiled *CompiledInsertQuery[T, R]
	input    T
	args     []interface{}
}

type fieldInfo struct {
	Name     string
	DbName   string
	Type     reflect.Type
	IsAuto   bool
	Position int
}

// Insert creates a new insert query builder
func Insert[T any](table string) *InsertBuilder[T] {
	inputType := reflect.TypeOf((*T)(nil)).Elem()
	fields := extractFields(inputType)

	return &InsertBuilder[T]{
		table:       table,
		inputType:   inputType,
		inputFields: fields,
	}
}

// Returning adds a returning clause to the insert query
func Returning[T, R any](ib *InsertBuilder[T]) *InsertReturningBuilder[T, R] {
	returningType := reflect.TypeOf((*R)(nil)).Elem()
	returningFields := extractFields(returningType)

	return &InsertReturningBuilder[T, R]{
		insert:          ib,
		returningType:   returningType,
		returningFields: returningFields,
	}
}

// Compile compiles the insert query into a reusable form
func (ib *InsertBuilder[T]) Compile() *CompiledInsertQuery[T, struct{}] {
	query := buildInsertQuery(ib.table, ib.inputFields, nil)

	return &CompiledInsertQuery[T, struct{}]{
		query:        query,
		inputFields:  ib.inputFields,
		hasReturning: false,
	}
}

// Compile compiles the insert with returning query into a reusable form
func (irb *InsertReturningBuilder[T, R]) Compile() *CompiledInsertQuery[T, R] {
	query := buildInsertQuery(irb.insert.table, irb.insert.inputFields, irb.returningFields)

	return &CompiledInsertQuery[T, R]{
		query:           query,
		inputFields:     irb.insert.inputFields,
		returningFields: irb.returningFields,
		hasReturning:    true,
	}
}

// New creates a new executable query with the given input
func (cq *CompiledInsertQuery[T, R]) New(input T) *ExecutableQuery[T, R] {
	args := extractArgs(input, cq.inputFields)

	return &ExecutableQuery[T, R]{
		compiled: cq,
		input:    input,
		args:     args,
	}
}

func (cq *CompiledInsertQuery[T, R]) PreviewQuery(input T) (string, []any) {
	args := extractArgs(input, cq.inputFields)
	return cq.query, args
}

// ExecContext executes the query and returns the result
func (eq *ExecutableQuery[T, R]) ExecContext(ctx context.Context, db DB) (R, error) {
	var result R

	if eq.compiled.hasReturning {
		row := db.QueryRowContext(ctx, eq.compiled.query, eq.args...)
		err := scanRow(row, &result, eq.compiled.returningFields)
		return result, err
	}

	// For queries without returning, just execute
	_, err := db.ExecContext(ctx, eq.compiled.query, eq.args...)
	return result, err
}

// Helper functions

func extractFields(t reflect.Type) []fieldInfo {
	var fields []fieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")

		if tag == "" || tag == "-" {
			continue
		}

		parts := strings.Split(tag, ",")
		dbName := parts[0]
		isAuto := false

		for _, part := range parts[1:] {
			if part == "auto" {
				isAuto = true
			}
		}

		fields = append(fields, fieldInfo{
			Name:     field.Name,
			DbName:   dbName,
			Type:     field.Type,
			IsAuto:   isAuto,
			Position: i,
		})
	}

	return fields
}

func buildInsertQuery(table string, inputFields, returningFields []fieldInfo) string {
	var insertFields []string
	var placeholders []string
	placeholderCount := 0

	for _, field := range inputFields {
		if !field.IsAuto {
			insertFields = append(insertFields, field.DbName)
			placeholderCount++
			placeholders = append(placeholders, fmt.Sprintf("$%d", placeholderCount))
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(insertFields, ", "),
		strings.Join(placeholders, ", "))

	if returningFields != nil && len(returningFields) > 0 {
		var returningCols []string
		for _, field := range returningFields {
			returningCols = append(returningCols, field.DbName)
		}
		query += " RETURNING " + strings.Join(returningCols, ", ")
	}

	return query
}

func extractArgs(input interface{}, fields []fieldInfo) []interface{} {
	v := reflect.ValueOf(input)
	var args []interface{}

	for _, field := range fields {
		if !field.IsAuto {
			fieldValue := v.FieldByName(field.Name)
			args = append(args, fieldValue.Interface())
		}
	}

	return args
}

func scanRow(row *sql.Row, dest interface{}, fields []fieldInfo) error {
	v := reflect.ValueOf(dest).Elem()
	var scanArgs []interface{}

	for _, field := range fields {
		fieldValue := v.FieldByName(field.Name)
		scanArgs = append(scanArgs, fieldValue.Addr().Interface())
	}

	return row.Scan(scanArgs...)
}
