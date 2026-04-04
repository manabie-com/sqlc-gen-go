package golang

import (
	"context"
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// TestGenerateDynamicFilter tests that the code generator produces correct output
// for queries with emit_dynamic_filter enabled.
func TestGenerateDynamicFilter(t *testing.T) {
	req := &plugin.GenerateRequest{
		SqlcVersion: "v1.0.0",
		Settings: &plugin.Settings{
			Engine: "postgresql",
		},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{
				{
					Name: "public",
					Tables: []*plugin.Table{
						{
							Rel: &plugin.Identifier{Schema: "public", Name: "items"},
							Columns: []*plugin.Column{
								{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}},
								{Name: "name", NotNull: true, Type: &plugin.Identifier{Name: "text"}},
								{Name: "status", NotNull: true, Type: &plugin.Identifier{Name: "text"}},
							},
						},
					},
				},
			},
		},
		Queries: []*plugin.Query{
			{
				Name:     "SearchItems",
				Cmd:      ":many",
				Filename: "query.sql",
				// Simulated SQL with :if annotations
				Text: "SELECT id, name, status FROM items\nWHERE name = $1\n  AND status = $2 -- :if @status\nORDER BY\n  id ASC -- :if @id_asc\n  id DESC -- :if @id_desc",
				Params: []*plugin.Parameter{
					{Number: 1, Column: &plugin.Column{Name: "name", NotNull: true, Type: &plugin.Identifier{Name: "text"}}},
					{Number: 2, Column: &plugin.Column{Name: "status", NotNull: true, Type: &plugin.Identifier{Name: "text"}}},
				},
				Columns: []*plugin.Column{
					{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}},
					{Name: "name", NotNull: true, Type: &plugin.Identifier{Name: "text"}},
					{Name: "status", NotNull: true, Type: &plugin.Identifier{Name: "text"}},
				},
			},
		},
		PluginOptions: []byte(`{
			"package": "testpkg",
			"sql_package": "pgx/v5",
			"emit_dynamic_filter": true
		}`),
		GlobalOptions: []byte(`{}`),
	}

	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Find the query file
	var queryFile, dynfilterFile string
	for _, f := range resp.Files {
		if f.Name == "query.sql.go" {
			queryFile = string(f.Contents)
		}
		if f.Name == "dynfilter.go" {
			dynfilterFile = string(f.Contents)
		}
	}

	if queryFile == "" {
		t.Fatal("query.sql.go not generated")
	}
	if dynfilterFile == "" {
		t.Fatal("dynfilter.go not generated")
	}

	// Verify params struct has pointer type for conditional param and bool fields for flags
	if !strings.Contains(queryFile, "*string") && !strings.Contains(queryFile, "Status *string") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected *string pointer type for conditional 'status' param")
	}
	if !strings.Contains(queryFile, "IdAsc") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected 'IdAsc' bool field in params struct")
	}
	if !strings.Contains(queryFile, "IdDesc") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected 'IdDesc' bool field in params struct")
	}

	// Verify DynamicSQL is called in the generated method
	if !strings.Contains(queryFile, "DynamicSQL(") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected DynamicSQL call in generated query method")
	}

	// Verify dynfilter.go contains the DynamicSQL function
	if !strings.Contains(dynfilterFile, "func DynamicSQL(") {
		t.Logf("dynfilter file:\n%s", dynfilterFile)
		t.Error("expected DynamicSQL function in dynfilter.go")
	}

	// Verify the SQL constant has -- :if $N markers
	if !strings.Contains(queryFile, "-- :if $") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected '-- :if $N' markers in SQL constant")
	}

	t.Logf("Generated query.sql.go:\n%s", queryFile)
	t.Logf("Generated dynfilter.go:\n%s", dynfilterFile)
}

func TestGenerateDynamicFilter_ValidationError(t *testing.T) {
	req := &plugin.GenerateRequest{
		SqlcVersion: "v1.0.0",
		Settings:    &plugin.Settings{Engine: "postgresql"},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas:       []*plugin.Schema{{Name: "public"}},
		},
		Queries:       []*plugin.Query{},
		PluginOptions: []byte(`{"package":"testpkg","sql_package":"pgx/v5","emit_dynamic_filter":true,"emit_prepared_queries":true}`),
		GlobalOptions: []byte(`{}`),
	}
	_, err := Generate(context.Background(), req)
	if err == nil {
		t.Error("expected error for emit_dynamic_filter + emit_prepared_queries")
	}
	if !strings.Contains(err.Error(), "emit_dynamic_filter") {
		t.Errorf("expected error to mention emit_dynamic_filter, got: %v", err)
	}
}

// TestGenerateDynamicFilter_FlagOnlyParams verifies that a query with only
// flag-only params (no WHERE $N predicates) generates a params struct with
// only bool fields and calls DynamicSQL correctly.
func TestGenerateDynamicFilter_FlagOnlyParams(t *testing.T) {
	req := &plugin.GenerateRequest{
		SqlcVersion: "v1.0.0",
		Settings:    &plugin.Settings{Engine: "postgresql"},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{{
				Name: "public",
				Tables: []*plugin.Table{{
					Rel:     &plugin.Identifier{Schema: "public", Name: "items"},
					Columns: []*plugin.Column{{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}}},
				}},
			}},
		},
		Queries: []*plugin.Query{{
			Name:     "ListItems",
			Cmd:      ":many",
			Filename: "query.sql",
			Text:     "SELECT id FROM items\nORDER BY\n  id ASC -- :if @sort_asc\n  id DESC -- :if @sort_desc",
			Params:   []*plugin.Parameter{},
			Columns:  []*plugin.Column{{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}}},
		}},
		PluginOptions: []byte(`{"package":"testpkg","sql_package":"pgx/v5","emit_dynamic_filter":true}`),
		GlobalOptions: []byte(`{}`),
	}

	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var queryFile string
	for _, f := range resp.Files {
		if f.Name == "query.sql.go" {
			queryFile = string(f.Contents)
		}
	}
	if queryFile == "" {
		t.Fatal("query.sql.go not generated")
	}

	if !strings.Contains(queryFile, "SortAsc") {
		t.Errorf("expected SortAsc bool field, got:\n%s", queryFile)
	}
	if !strings.Contains(queryFile, "SortDesc") {
		t.Errorf("expected SortDesc bool field, got:\n%s", queryFile)
	}
	if !strings.Contains(queryFile, "DynamicSQL(") {
		t.Errorf("expected DynamicSQL call, got:\n%s", queryFile)
	}
	t.Logf("Generated:\n%s", queryFile)
}

// TestGenerateDynamicFilter_BlockAnnotation verifies that a standalone
// block-style annotation (-- :if @flag on its own line) causes the next line
// to be skipped when the flag is false, and the marker appears in the SQL constant.
func TestGenerateDynamicFilter_BlockAnnotation(t *testing.T) {
	req := &plugin.GenerateRequest{
		SqlcVersion: "v1.0.0",
		Settings:    &plugin.Settings{Engine: "postgresql"},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{{
				Name: "public",
				Tables: []*plugin.Table{{
					Rel: &plugin.Identifier{Schema: "public", Name: "items"},
					Columns: []*plugin.Column{
						{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}},
						{Name: "status", Type: &plugin.Identifier{Name: "text"}},
					},
				}},
			}},
		},
		Queries: []*plugin.Query{{
			Name:     "GetItems",
			Cmd:      ":many",
			Filename: "query.sql",
			Text:     "SELECT id, status FROM items\nWHERE id = $1\n-- :if @filter_status\n  AND status = $2",
			Params: []*plugin.Parameter{
				{Number: 1, Column: &plugin.Column{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}}},
				{Number: 2, Column: &plugin.Column{Name: "status", Type: &plugin.Identifier{Name: "text"}}},
			},
			Columns: []*plugin.Column{
				{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}},
				{Name: "status", Type: &plugin.Identifier{Name: "text"}},
			},
		}},
		PluginOptions: []byte(`{"package":"testpkg","sql_package":"pgx/v5","emit_dynamic_filter":true}`),
		GlobalOptions: []byte(`{}`),
	}

	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var queryFile string
	for _, f := range resp.Files {
		if f.Name == "query.sql.go" {
			queryFile = string(f.Contents)
		}
	}
	if queryFile == "" {
		t.Fatal("query.sql.go not generated")
	}

	// The SQL constant should have a standalone :if marker for the block annotation.
	if !strings.Contains(queryFile, "-- :if $") {
		t.Errorf("expected :if $N marker in SQL constant, got:\n%s", queryFile)
	}
	// filter_status is a flag-only param → bool field, not a pointer.
	if !strings.Contains(queryFile, "FilterStatus") {
		t.Errorf("expected FilterStatus bool field, got:\n%s", queryFile)
	}
	// status is NOT conditional (the block flag controls skipping, not the param itself),
	// so it should remain a plain type, not a pointer.
	if strings.Contains(queryFile, "*pgtype.Text") || strings.Contains(queryFile, "Status *") {
		t.Errorf("status should NOT be a pointer in block-annotation mode, got:\n%s", queryFile)
	}
	t.Logf("Generated:\n%s", queryFile)
}

