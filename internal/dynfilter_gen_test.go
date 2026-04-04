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

	// Verify the SQL constant has :dynif markers
	if !strings.Contains(queryFile, ":dynif") {
		t.Logf("query file:\n%s", queryFile)
		t.Error("expected :dynif markers in SQL constant")
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

