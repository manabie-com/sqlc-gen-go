package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

// makeEngineReq creates a GenerateRequest with the given engine.
func makeEngineReq(engine string) *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog:  &plugin.Catalog{DefaultSchema: "public", Schemas: []*plugin.Schema{}},
		Settings: &plugin.Settings{Engine: engine},
	}
}

// parseTestOpts parses Options from a JSON string for testing.
func parseTestOpts(t *testing.T, pluginJSON string) (*plugin.GenerateRequest, *opts.Options) {
	t.Helper()
	req := makeEngineReq("postgresql")
	req.PluginOptions = []byte(pluginJSON)
	o, err := opts.Parse(req)
	if err != nil {
		t.Fatalf("opts.Parse: %v", err)
	}
	return req, o
}

func TestGoType(t *testing.T) {
	t.Run("scalar_type", func(t *testing.T) {
		req := makeEngineReq("postgresql")
		o := &opts.Options{SqlPackage: "pgx/v5"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "text"}, NotNull: true}
		if got := goType(req, o, col); got != "string" {
			t.Errorf("goType(text, notNull) = %q, want string", got)
		}
	})

	t.Run("sqlc_slice", func(t *testing.T) {
		req := makeEngineReq("postgresql")
		o := &opts.Options{SqlPackage: "pgx/v5"}
		col := &plugin.Column{
			Type: &plugin.Identifier{Name: "integer"}, NotNull: true, IsSqlcSlice: true,
		}
		if got := goType(req, o, col); got != "[]int32" {
			t.Errorf("goType(integer, sqlcSlice) = %q, want []int32", got)
		}
	})

	t.Run("array_single_dim", func(t *testing.T) {
		req := makeEngineReq("postgresql")
		o := &opts.Options{SqlPackage: "pgx/v5"}
		col := &plugin.Column{
			Type: &plugin.Identifier{Name: "integer"}, IsArray: true, ArrayDims: 1,
		}
		if got := goType(req, o, col); got != "[]int32" {
			t.Errorf("goType(integer, array dims=1) = %q, want []int32", got)
		}
	})

	t.Run("array_multi_dim", func(t *testing.T) {
		req := makeEngineReq("postgresql")
		o := &opts.Options{SqlPackage: "pgx/v5"}
		col := &plugin.Column{
			Type: &plugin.Identifier{Name: "integer"}, IsArray: true, ArrayDims: 2,
		}
		if got := goType(req, o, col); got != "[][]int32" {
			t.Errorf("goType(integer, array dims=2) = %q, want [][]int32", got)
		}
	})

	t.Run("column_override_by_name", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.email","go_type":"github.com/example/types.Email"}]}`)
		col := &plugin.Column{
			Name: "email", Type: &plugin.Identifier{Name: "text"}, NotNull: true,
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		if got := goType(req, o, col); got != "types.Email" {
			t.Errorf("goType with column override = %q, want types.Email", got)
		}
	})

	t.Run("column_override_with_sqlc_slice", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.tags","go_type":"string"}]}`)
		col := &plugin.Column{
			Name: "tags", Type: &plugin.Identifier{Name: "text"}, IsSqlcSlice: true,
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		if got := goType(req, o, col); got != "[]string" {
			t.Errorf("goType with column override + sqlcSlice = %q, want []string", got)
		}
	})

	t.Run("column_override_uses_original_name", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.orig_email","go_type":"github.com/example/types.Email"}]}`)
		col := &plugin.Column{
			Name: "email", OriginalName: "orig_email",
			Type: &plugin.Identifier{Name: "text"}, NotNull: true,
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		if got := goType(req, o, col); got != "types.Email" {
			t.Errorf("goType using OriginalName for override = %q, want types.Email", got)
		}
	})
}

func TestGoInnerType(t *testing.T) {
	t.Run("package_override_matches_not_null", func(t *testing.T) {
		// nullable=false override applies when notNull=true
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"db_type":"text","nullable":false,"go_type":"github.com/example/types.MyString"}]}`)
		req.Settings = &plugin.Settings{Engine: "postgresql"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "text"}, NotNull: true}
		if got := goInnerType(req, o, col); got != "types.MyString" {
			t.Errorf("goInnerType with db_type override (notNull=true) = %q, want types.MyString", got)
		}
	})

	t.Run("package_override_no_match_wrong_nullability", func(t *testing.T) {
		// nullable=false override does NOT apply when column is nullable (notNull=false)
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"db_type":"text","nullable":false,"go_type":"github.com/example/types.MyString"}]}`)
		req.Settings = &plugin.Settings{Engine: "postgresql"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "text"}, NotNull: false}
		if got := goInnerType(req, o, col); got != "pgtype.Text" {
			t.Errorf("goInnerType without match = %q, want pgtype.Text", got)
		}
	})

	t.Run("mysql_engine", func(t *testing.T) {
		req := makeEngineReq("mysql")
		o := &opts.Options{SqlPackage: "database/sql"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "varchar"}, NotNull: true}
		if got := goInnerType(req, o, col); got != "string" {
			t.Errorf("goInnerType(mysql, varchar) = %q, want string", got)
		}
	})

	t.Run("sqlite_engine", func(t *testing.T) {
		req := makeEngineReq("sqlite")
		o := &opts.Options{SqlPackage: "database/sql"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "integer"}, NotNull: true}
		if got := goInnerType(req, o, col); got != "int64" {
			t.Errorf("goInnerType(sqlite, integer) = %q, want int64", got)
		}
	})

	t.Run("unknown_engine_returns_interface", func(t *testing.T) {
		req := makeEngineReq("unknown")
		o := &opts.Options{SqlPackage: "database/sql"}
		col := &plugin.Column{Type: &plugin.Identifier{Name: "text"}, NotNull: true}
		if got := goInnerType(req, o, col); got != "interface{}" {
			t.Errorf("goInnerType(unknown engine) = %q, want interface{}", got)
		}
	})
}

func TestAddExtraGoStructTags(t *testing.T) {
	t.Run("adds_tags_on_matching_column", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.name","go_type":"string","go_struct_tag":"validate:\"required\""}]}`)
		col := &plugin.Column{
			Name: "name", Type: &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if tags["validate"] != "required" {
			t.Errorf("expected validate=required, got %v", tags)
		}
	})

	t.Run("no_tags_different_table", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.name","go_type":"string","go_struct_tag":"validate:\"required\""}]}`)
		col := &plugin.Column{
			Name: "name", Type: &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "orders"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if len(tags) != 0 {
			t.Errorf("expected no tags for different table, got %v", tags)
		}
	})

	t.Run("no_tags_different_column", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.email","go_type":"string","go_struct_tag":"validate:\"email\""}]}`)
		col := &plugin.Column{
			Name: "name", Type: &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if len(tags) != 0 {
			t.Errorf("expected no tags for different column, got %v", tags)
		}
	})

	t.Run("no_tags_when_override_has_no_struct_tag", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.name","go_type":"string"}]}`)
		col := &plugin.Column{
			Name: "name", Type: &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if len(tags) != 0 {
			t.Errorf("expected no tags when override has no go_struct_tag, got %v", tags)
		}
	})

	t.Run("multiple_tags_from_single_override", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.name","go_type":"string","go_struct_tag":"validate:\"required\" form:\"name\""}]}`)
		col := &plugin.Column{
			Name: "name", Type: &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if tags["validate"] != "required" || tags["form"] != "name" {
			t.Errorf("expected both validate and form tags, got %v", tags)
		}
	})

	t.Run("uses_original_name_for_matching", func(t *testing.T) {
		req, o := parseTestOpts(t, `{"package":"db","sql_package":"pgx/v5","overrides":[{"column":"users.original_col","go_type":"string","go_struct_tag":"binding:\"required\""}]}`)
		col := &plugin.Column{
			Name: "col", OriginalName: "original_col",
			Type:  &plugin.Identifier{Name: "text"},
			Table: &plugin.Identifier{Schema: "public", Name: "users"},
		}
		tags := map[string]string{}
		addExtraGoStructTags(tags, req, o, col)
		if tags["binding"] != "required" {
			t.Errorf("expected binding tag using OriginalName, got %v", tags)
		}
	})
}
