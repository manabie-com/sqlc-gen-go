package opts

import (
	"encoding/json"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func makeReq(pluginOpts any) *plugin.GenerateRequest {
	b, _ := json.Marshal(pluginOpts)
	return &plugin.GenerateRequest{PluginOptions: b}
}

func TestParseOpts_DefaultValues(t *testing.T) {
	req := makeReq(map[string]any{"package": "db"})
	opts, err := Parse(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Package != "db" {
		t.Errorf("package = %q, want %q", opts.Package, "db")
	}
	if opts.QueryParameterLimit == nil || *opts.QueryParameterLimit != 1 {
		t.Errorf("QueryParameterLimit should default to 1")
	}
	if opts.Initialisms == nil {
		t.Errorf("Initialisms should not be nil")
	}
	if _, ok := opts.InitialismsMap["id"]; !ok {
		t.Errorf("InitialismsMap should contain 'id' by default")
	}
}

func TestParseOpts_EmptyOptions(t *testing.T) {
	req := &plugin.GenerateRequest{}
	opts, err := Parse(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts == nil {
		t.Fatal("opts should not be nil")
	}
}

func TestParseOpts_PackageFromOut(t *testing.T) {
	req := makeReq(map[string]any{"out": "./internal/db"})
	opts, err := Parse(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Package != "db" {
		t.Errorf("package from out = %q, want %q", opts.Package, "db")
	}
}

func TestParseOpts_MissingPackage(t *testing.T) {
	req := makeReq(map[string]any{"emit_interface": true})
	_, err := Parse(req)
	if err == nil {
		t.Error("expected error for missing package name")
	}
}

func TestParseOpts_InvalidJSON(t *testing.T) {
	req := &plugin.GenerateRequest{PluginOptions: []byte("{bad json")}
	_, err := Parse(req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseOpts_InvalidSQLPackage(t *testing.T) {
	req := makeReq(map[string]any{"package": "db", "sql_package": "unknown/pkg"})
	_, err := Parse(req)
	if err == nil {
		t.Error("expected error for invalid sql_package")
	}
}

func TestParseOpts_InvalidSQLDriver(t *testing.T) {
	req := makeReq(map[string]any{"package": "db", "sql_driver": "unknown/driver"})
	_, err := Parse(req)
	if err == nil {
		t.Error("expected error for invalid sql_driver")
	}
}

func TestParseOpts_ValidSQLPackageAndDriver(t *testing.T) {
	req := makeReq(map[string]any{
		"package":     "db",
		"sql_package": "pgx/v5",
		"sql_driver":  "github.com/jackc/pgx/v5",
	})
	_, err := Parse(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseOpts_GlobalOverrides(t *testing.T) {
	global, _ := json.Marshal(map[string]any{
		"overrides": []map[string]any{
			{"db_type": "uuid", "go_type": "string"},
		},
	})
	req := makeReq(map[string]any{"package": "db"})
	req.GlobalOptions = global
	opts, err := Parse(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts.Overrides) != 1 {
		t.Errorf("expected 1 global override, got %d", len(opts.Overrides))
	}
}

func TestValidateOpts_MutuallyExclusive(t *testing.T) {
	limit := int32(1)
	cases := []struct {
		name string
		opts *Options
	}{
		{
			"emit_methods_with_db_argument + emit_prepared_queries",
			&Options{EmitMethodsWithDbArgument: true, EmitPreparedQueries: true, QueryParameterLimit: &limit},
		},
		{
			"emit_per_file_queries + emit_prepared_queries",
			&Options{EmitPerFileQueries: true, EmitPreparedQueries: true, QueryParameterLimit: &limit},
		},
		{
			"emit_dynamic_filter + emit_prepared_queries",
			&Options{EmitDynamicFilter: true, EmitPreparedQueries: true, QueryParameterLimit: &limit},
		},
		{
			"disable_result_slice_pointers without emit_result_struct_pointers",
			&Options{DisableResultSlicePointers: true, QueryParameterLimit: &limit},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOpts(tc.opts); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestValidateOpts_NegativeQueryParameterLimit(t *testing.T) {
	limit := int32(-1)
	err := ValidateOpts(&Options{QueryParameterLimit: &limit})
	if err == nil {
		t.Error("expected error for negative QueryParameterLimit")
	}
}

func TestValidateOpts_Valid(t *testing.T) {
	limit := int32(1)
	err := ValidateOpts(&Options{QueryParameterLimit: &limit})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOpts_DisableResultSlicePointersWithEmit(t *testing.T) {
	limit := int32(1)
	err := ValidateOpts(&Options{
		EmitResultStructPointers:   true,
		DisableResultSlicePointers: true,
		QueryParameterLimit:        &limit,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
