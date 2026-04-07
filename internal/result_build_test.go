package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

// minimalOpts returns a bare-minimum Options suitable for result tests.
func minimalOpts() *opts.Options {
	limit := int32(1)
	return &opts.Options{
		SqlPackage:          "pgx/v5",
		InitialismsMap:      map[string]struct{}{"id": {}},
		QueryParameterLimit: &limit,
	}
}

// pgReq returns a GenerateRequest with a PostgreSQL engine and the given schemas.
func pgReq(schemas ...*plugin.Schema) *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: schemas},
		Settings: &plugin.Settings{Engine: "postgresql"},
	}
}

func TestBuildEnums(t *testing.T) {
	t.Run("empty_schemas", func(t *testing.T) {
		enums := buildEnums(pgReq(), minimalOpts())
		if len(enums) != 0 {
			t.Errorf("expected 0 enums, got %d", len(enums))
		}
	})

	t.Run("basic_enum_in_default_schema", func(t *testing.T) {
		req := pgReq(&plugin.Schema{
			Name:  "public",
			Enums: []*plugin.Enum{{Name: "status", Comment: "user status", Vals: []string{"active", "inactive", "pending"}}},
		})
		enums := buildEnums(req, minimalOpts())
		if len(enums) != 1 {
			t.Fatalf("expected 1 enum, got %d", len(enums))
		}
		e := enums[0]
		if e.Name != "Status" {
			t.Errorf("Name = %q, want Status", e.Name)
		}
		if e.Comment != "user status" {
			t.Errorf("Comment = %q, want 'user status'", e.Comment)
		}
		if len(e.Constants) != 3 || e.Constants[0].Name != "StatusActive" {
			t.Errorf("unexpected constants: %v", e.Constants)
		}
	})

	t.Run("skips_pg_catalog_and_information_schema", func(t *testing.T) {
		req := pgReq(
			&plugin.Schema{Name: "pg_catalog", Enums: []*plugin.Enum{{Name: "pg_enum", Vals: []string{"a"}}}},
			&plugin.Schema{Name: "information_schema", Enums: []*plugin.Enum{{Name: "info_enum", Vals: []string{"b"}}}},
			&plugin.Schema{Name: "public", Enums: []*plugin.Enum{{Name: "status", Vals: []string{"active"}}}},
		)
		enums := buildEnums(req, minimalOpts())
		if len(enums) != 1 || enums[0].Name != "Status" {
			t.Errorf("expected only public.status, got %v", enums)
		}
	})

	t.Run("non_default_schema_prefixes_name", func(t *testing.T) {
		req := pgReq(&plugin.Schema{
			Name:  "myschema",
			Enums: []*plugin.Enum{{Name: "role", Vals: []string{"admin"}}},
		})
		enums := buildEnums(req, minimalOpts())
		if len(enums) != 1 || enums[0].Name != "MyschemaRole" {
			t.Errorf("expected MyschemaRole, got %v", enums)
		}
	})

	t.Run("duplicate_vals_get_unique_names", func(t *testing.T) {
		req := pgReq(&plugin.Schema{
			Name:  "public",
			Enums: []*plugin.Enum{{Name: "myenum", Vals: []string{"dup", "dup", "other"}}},
		})
		enums := buildEnums(req, minimalOpts())
		if len(enums[0].Constants) != 3 {
			t.Errorf("expected 3 constants, got %d", len(enums[0].Constants))
		}
		names := map[string]bool{}
		for _, c := range enums[0].Constants {
			if names[c.Name] {
				t.Errorf("duplicate constant name %q", c.Name)
			}
			names[c.Name] = true
		}
	})

	t.Run("json_tags_emitted_when_enabled", func(t *testing.T) {
		req := pgReq(&plugin.Schema{
			Name:  "public",
			Enums: []*plugin.Enum{{Name: "status", Vals: []string{"active"}}},
		})
		o := minimalOpts()
		o.EmitJsonTags = true
		enums := buildEnums(req, o)
		if enums[0].NameTags["json"] == "" || enums[0].ValidTags["json"] == "" {
			t.Error("expected json tags on enum when EmitJsonTags=true")
		}
	})

	t.Run("sorted_by_name", func(t *testing.T) {
		req := pgReq(&plugin.Schema{
			Name: "public",
			Enums: []*plugin.Enum{
				{Name: "zebra", Vals: []string{"z"}},
				{Name: "apple", Vals: []string{"a"}},
				{Name: "mango", Vals: []string{"m"}},
			},
		})
		enums := buildEnums(req, minimalOpts())
		if len(enums) != 3 || enums[0].Name != "Apple" || enums[2].Name != "Zebra" {
			t.Errorf("enums not sorted: %v", enums)
		}
	})
}

func TestNewGoEmbed(t *testing.T) {
	structs := []Struct{
		{
			Name:   "User",
			Table:  &plugin.Identifier{Schema: "public", Name: "users"},
			Fields: []Field{{Name: "ID", Type: "int32"}, {Name: "Name", Type: "string"}},
		},
		{
			Name:  "Order",
			Table: &plugin.Identifier{Schema: "myschema", Name: "orders"},
		},
	}

	t.Run("nil_embed_returns_nil", func(t *testing.T) {
		if got := newGoEmbed(nil, structs, "public"); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("no_match_returns_nil", func(t *testing.T) {
		embed := &plugin.Identifier{Schema: "public", Name: "products"}
		if got := newGoEmbed(embed, structs, "public"); got != nil {
			t.Errorf("expected nil for non-matching embed, got %+v", got)
		}
	})

	t.Run("matches_by_schema_and_name", func(t *testing.T) {
		embed := &plugin.Identifier{Schema: "public", Name: "users"}
		got := newGoEmbed(embed, structs, "public")
		if got == nil {
			t.Fatal("expected non-nil for matching embed")
		}
		if got.modelType != "User" || got.modelName != "User" {
			t.Errorf("modelType/Name = %q/%q, want User/User", got.modelType, got.modelName)
		}
		if len(got.fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(got.fields))
		}
	})

	t.Run("empty_schema_uses_default_schema", func(t *testing.T) {
		embed := &plugin.Identifier{Name: "users"} // no schema
		got := newGoEmbed(embed, structs, "public")
		if got == nil {
			t.Error("expected match when embed schema is empty (falls back to defaultSchema)")
		}
	})

	t.Run("explicit_non_default_schema", func(t *testing.T) {
		embed := &plugin.Identifier{Schema: "myschema", Name: "orders"}
		got := newGoEmbed(embed, structs, "public")
		if got == nil || got.modelType != "Order" {
			t.Errorf("expected Order embed, got %v", got)
		}
	})
}

func TestColumnsToStruct(t *testing.T) {
	req := pgReq()

	t.Run("basic_fields", func(t *testing.T) {
		o := minimalOpts()
		cols := []goColumn{
			{id: 0, Column: &plugin.Column{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
			{id: 1, Column: &plugin.Column{Name: "name", Type: &plugin.Identifier{Name: "text"}, NotNull: true}},
		}
		gs, err := columnsToStruct(req, o, "UserRow", cols, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gs.Name != "UserRow" || len(gs.Fields) != 2 {
			t.Errorf("unexpected struct: name=%q fields=%d", gs.Name, len(gs.Fields))
		}
		if gs.Fields[0].Name != "ID" || gs.Fields[1].Name != "Name" {
			t.Errorf("unexpected field names: %v", gs.Fields)
		}
	})

	t.Run("duplicate_column_names_get_suffix", func(t *testing.T) {
		o := minimalOpts()
		cols := []goColumn{
			{id: 0, Column: &plugin.Column{Name: "count", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
			{id: 1, Column: &plugin.Column{Name: "count", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
		}
		gs, err := columnsToStruct(req, o, "CountRow", cols, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gs.Fields[1].Name != "Count_2" {
			t.Errorf("duplicate field name = %q, want Count_2", gs.Fields[1].Name)
		}
	})

	t.Run("db_tags_emitted", func(t *testing.T) {
		o := minimalOpts()
		o.EmitDbTags = true
		cols := []goColumn{
			{id: 0, Column: &plugin.Column{Name: "user_id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
		}
		gs, _ := columnsToStruct(req, o, "Row", cols, false)
		if gs.Fields[0].Tags["db"] != "user_id" {
			t.Errorf("db tag = %q, want user_id", gs.Fields[0].Tags["db"])
		}
	})

	t.Run("json_tags_emitted", func(t *testing.T) {
		o := minimalOpts()
		o.EmitJsonTags = true
		cols := []goColumn{
			{id: 0, Column: &plugin.Column{Name: "user_id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
		}
		gs, _ := columnsToStruct(req, o, "Row", cols, false)
		if gs.Fields[0].Tags["json"] == "" {
			t.Error("expected json tag")
		}
	})

	t.Run("embed_column_uses_model_type", func(t *testing.T) {
		o := minimalOpts()
		embed := &goEmbed{modelType: "User", modelName: "User", fields: []Field{{Name: "ID", Type: "int32"}}}
		cols := []goColumn{
			{id: 0, Column: &plugin.Column{Name: "user", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}, embed: embed},
		}
		gs, err := columnsToStruct(req, o, "UserRow", cols, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gs.Fields[0].Type != "User" || len(gs.Fields[0].EmbedFields) != 1 {
			t.Errorf("embed field: Type=%q EmbedFields=%d", gs.Fields[0].Type, len(gs.Fields[0].EmbedFields))
		}
	})

	t.Run("incompatible_named_param_types_returns_error", func(t *testing.T) {
		o := minimalOpts()
		cols := []goColumn{
			{id: 1, Column: &plugin.Column{Name: "val", Type: &plugin.Identifier{Name: "integer"}, NotNull: true, IsNamedParam: true}},
			{id: 1, Column: &plugin.Column{Name: "val", Type: &plugin.Identifier{Name: "text"}, NotNull: true, IsNamedParam: true}},
		}
		_, err := columnsToStruct(req, o, "Row", cols, true)
		if err == nil {
			t.Error("expected error for incompatible named param types")
		}
	})
}

func TestBuildQueries(t *testing.T) {
	t.Run("empty_input", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog: &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries:  []*plugin.Query{},
		}
		qs, err := buildQueries(req, minimalOpts(), nil)
		if err != nil || len(qs) != 0 {
			t.Errorf("expected empty result, got err=%v len=%d", err, len(qs))
		}
	})

	t.Run("skips_queries_with_empty_name_or_cmd", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{
				{Name: "", Cmd: metadata.CmdOne, Text: "SELECT 1"},
				{Name: "NoCmd", Cmd: "", Text: "SELECT 1"},
			},
		}
		qs, err := buildQueries(req, minimalOpts(), nil)
		if err != nil || len(qs) != 0 {
			t.Errorf("expected both skipped, got err=%v len=%d", err, len(qs))
		}
	})

	t.Run("exec_query_single_param", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{{
				Name: "DeleteUser", Cmd: metadata.CmdExec,
				Text: "DELETE FROM users WHERE id = $1",
				Params: []*plugin.Parameter{
					{Number: 1, Column: &plugin.Column{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
				},
			}},
		}
		qs, err := buildQueries(req, minimalOpts(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(qs) != 1 || qs[0].MethodName != "DeleteUser" {
			t.Errorf("unexpected queries: %v", qs)
		}
	})

	t.Run("one_query_single_column_return", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{{
				Name: "GetID", Cmd: metadata.CmdOne, Text: "SELECT id FROM users WHERE id = $1",
				Params:  []*plugin.Parameter{{Number: 1, Column: &plugin.Column{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}}},
				Columns: []*plugin.Column{{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
			}},
		}
		qs, err := buildQueries(req, minimalOpts(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qs[0].Ret.DBName != "id" {
			t.Errorf("ret.DBName = %q, want id", qs[0].Ret.DBName)
		}
	})

	t.Run("multiple_params_exceed_limit_produce_struct", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{{
				Name: "CreateUser", Cmd: metadata.CmdExec, Text: "INSERT INTO users (name, email) VALUES ($1, $2)",
				Params: []*plugin.Parameter{
					{Number: 1, Column: &plugin.Column{Name: "name", Type: &plugin.Identifier{Name: "text"}, NotNull: true}},
					{Number: 2, Column: &plugin.Column{Name: "email", Type: &plugin.Identifier{Name: "text"}, NotNull: true}},
				},
			}},
		}
		qs, err := buildQueries(req, minimalOpts(), nil) // limit=1, so 2 params → struct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qs[0].Arg.Struct == nil {
			t.Error("expected Arg.Struct for multiple params")
		}
	})

	t.Run("reuses_matching_existing_struct", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{{
				Name: "GetUser", Cmd: metadata.CmdOne, Text: "SELECT id, name FROM users WHERE id = $1",
				Params: []*plugin.Parameter{
					{Number: 1, Column: &plugin.Column{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true}},
				},
				Columns: []*plugin.Column{
					{Name: "id", Type: &plugin.Identifier{Name: "integer"}, NotNull: true, Table: &plugin.Identifier{Schema: "public", Name: "users"}},
					{Name: "name", Type: &plugin.Identifier{Name: "text"}, NotNull: true, Table: &plugin.Identifier{Schema: "public", Name: "users"}},
				},
			}},
		}
		existing := Struct{
			Name:   "User",
			Table:  &plugin.Identifier{Schema: "public", Name: "users"},
			Fields: []Field{{Name: "ID", Type: "int32"}, {Name: "Name", Type: "string"}},
		}
		qs, err := buildQueries(req, minimalOpts(), []Struct{existing})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qs[0].Ret.Emit || qs[0].Ret.Struct.Name != "User" {
			t.Errorf("expected reuse of User struct (Emit=false), got Emit=%v Name=%q", qs[0].Ret.Emit, qs[0].Ret.Struct.Name)
		}
	})

	t.Run("emit_sql_as_comment", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries: []*plugin.Query{{
				Name: "ListUsers", Cmd: metadata.CmdMany, Text: "SELECT id FROM users",
			}},
		}
		o := minimalOpts()
		o.EmitSqlAsComment = true
		qs, err := buildQueries(req, o, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := false
		for _, c := range qs[0].Comments {
			if c == "  SELECT id FROM users" {
				found = true
			}
		}
		if !found {
			t.Errorf("SQL text not found in comments: %v", qs[0].Comments)
		}
	})

	t.Run("exported_query_names", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
			Settings: &plugin.Settings{Engine: "postgresql"},
			Queries:  []*plugin.Query{{Name: "getUser", Cmd: metadata.CmdOne, Text: "SELECT 1"}},
		}
		o := minimalOpts()
		o.EmitExportedQueries = true
		qs, err := buildQueries(req, o, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qs[0].ConstantName != "GetUser" {
			t.Errorf("ConstantName = %q, want GetUser", qs[0].ConstantName)
		}
	})
}
