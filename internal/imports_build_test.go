package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

// baseImp creates an importer with the given sql package.
func baseImp(sqlPkg string) *importer {
	limit := int32(1)
	return &importer{
		Options: &opts.Options{
			SqlPackage:          sqlPkg,
			InitialismsMap:      map[string]struct{}{"id": {}},
			QueryParameterLimit: &limit,
		},
	}
}

func TestBuildImports(t *testing.T) {
	t.Run("sql_null_adds_database_sql", func(t *testing.T) {
		o := baseImp("database/sql").Options
		std, _ := buildImports(o, nil, func(name string) bool { return name == "sql.Null" })
		if _, ok := std["database/sql"]; !ok {
			t.Error("expected database/sql for sql.Null usage")
		}
	})

	t.Run("exec_result_pgxv5_adds_pgconn", func(t *testing.T) {
		o := baseImp("pgx/v5").Options
		_, pkg := buildImports(o, []Query{{Cmd: metadata.CmdExecResult}}, func(string) bool { return false })
		found := false
		for s := range pkg {
			if s.Path == "github.com/jackc/pgx/v5/pgconn" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgx/v5/pgconn for ExecResult with pgx/v5")
		}
	})

	t.Run("exec_result_pgxv4_adds_pgconn", func(t *testing.T) {
		o := baseImp("pgx/v4").Options
		_, pkg := buildImports(o, []Query{{Cmd: metadata.CmdExecResult}}, func(string) bool { return false })
		found := false
		for s := range pkg {
			if s.Path == "github.com/jackc/pgconn" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgconn for ExecResult with pgx/v4")
		}
	})

	t.Run("exec_result_stdlib_adds_database_sql", func(t *testing.T) {
		o := baseImp("database/sql").Options
		std, _ := buildImports(o, []Query{{Cmd: metadata.CmdExecResult}}, func(string) bool { return false })
		if _, ok := std["database/sql"]; !ok {
			t.Error("expected database/sql for ExecResult with stdlib")
		}
	})

	t.Run("pgtype_pgxv5_adds_pgx_pgtype", func(t *testing.T) {
		o := baseImp("pgx/v5").Options
		_, pkg := buildImports(o, nil, func(name string) bool { return name == "pgtype." })
		found := false
		for s := range pkg {
			if s.Path == "github.com/jackc/pgx/v5/pgtype" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgx/v5/pgtype when pgtype. used with pgx/v5")
		}
	})

	t.Run("pgtype_pgxv4_adds_pgtype", func(t *testing.T) {
		o := baseImp("pgx/v4").Options
		_, pkg := buildImports(o, nil, func(name string) bool { return name == "pgtype." })
		found := false
		for s := range pkg {
			if s.Path == "github.com/jackc/pgtype" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgtype for pgx/v4")
		}
	})

	t.Run("uuid_adds_google_uuid", func(t *testing.T) {
		o := baseImp("database/sql").Options
		for _, typName := range []string{"uuid.UUID", "uuid.NullUUID"} {
			_, pkg := buildImports(o, nil, func(name string) bool { return name == typName })
			found := false
			for s := range pkg {
				if s.Path == "github.com/google/uuid" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected uuid import for %q", typName)
			}
		}
	})

	t.Run("pgvector_adds_pgvector_go", func(t *testing.T) {
		o := baseImp("pgx/v5").Options
		_, pkg := buildImports(o, nil, func(name string) bool { return name == "pgvector.Vector" })
		found := false
		for s := range pkg {
			if s.Path == "github.com/pgvector/pgvector-go" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgvector import")
		}
	})

	t.Run("pqtype_adds_sqlc_pqtype", func(t *testing.T) {
		o := baseImp("database/sql").Options
		_, pkg := buildImports(o, nil, func(name string) bool { return name == "pqtype.NullRawMessage" })
		found := false
		for s := range pkg {
			if s.Path == "github.com/sqlc-dev/pqtype" {
				found = true
			}
		}
		if !found {
			t.Error("expected pqtype import")
		}
	})

	t.Run("stdlib_type_time_adds_time_package", func(t *testing.T) {
		o := baseImp("database/sql").Options
		std, _ := buildImports(o, nil, func(name string) bool { return name == "time.Time" })
		if _, ok := std["time"]; !ok {
			t.Error("expected time package for time.Time")
		}
	})
}

func TestModelImports(t *testing.T) {
	t.Run("enums_add_fmt_and_driver", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Enums = []Enum{{Name: "Status"}}
		out := imp.modelImports()
		hasFmt, hasDriver := false, false
		for _, s := range out.Std {
			switch s.Path {
			case "fmt":
				hasFmt = true
			case "database/sql/driver":
				hasDriver = true
			}
		}
		if !hasFmt || !hasDriver {
			t.Errorf("missing imports: fmt=%v driver=%v", hasFmt, hasDriver)
		}
	})

	t.Run("time_struct_field_adds_time_package", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Structs = []Struct{{Fields: []Field{{Type: "time.Time"}}}}
		out := imp.modelImports()
		found := false
		for _, s := range out.Std {
			if s.Path == "time" {
				found = true
			}
		}
		if !found {
			t.Error("expected time package for time.Time struct field")
		}
	})

	t.Run("no_types_no_enums_no_imports", func(t *testing.T) {
		imp := baseImp("database/sql")
		out := imp.modelImports()
		if len(out.Std) != 0 || len(out.Dep) != 0 {
			t.Errorf("expected empty imports, got std=%v dep=%v", out.Std, out.Dep)
		}
	})
}

func TestQueryImports(t *testing.T) {
	t.Run("non_copyfrom_query_adds_context", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Queries = []Query{{Cmd: metadata.CmdOne, SourceName: "users.sql", SQL: "SELECT 1"}}
		out := imp.queryImports("users.sql")
		found := false
		for _, s := range out.Std {
			if s.Path == "context" {
				found = true
			}
		}
		if !found {
			t.Error("expected context import for non-CopyFrom query")
		}
	})

	t.Run("different_file_no_imports", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Queries = []Query{{Cmd: metadata.CmdOne, SourceName: "users.sql"}}
		out := imp.queryImports("orders.sql")
		for _, s := range out.Std {
			if s.Path == "context" {
				t.Error("unexpected context for file with no matching queries")
			}
		}
	})

	t.Run("batch_queries_skipped_for_file_imports", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{
			{Cmd: metadata.CmdBatchExec, SourceName: "users.sql"},
			{Cmd: metadata.CmdOne, SourceName: "users.sql", SQL: "SELECT 1"},
		}
		out := imp.queryImports("users.sql")
		found := false
		for _, s := range out.Std {
			if s.Path == "context" {
				found = true
			}
		}
		if !found {
			t.Error("expected context for non-batch query in file even when batch query present")
		}
	})

	t.Run("array_ret_field_stdlib_adds_lib_pq", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Queries = []Query{{
			Cmd: metadata.CmdMany, SourceName: "tags.sql",
			Ret: QueryValue{Struct: &Struct{Fields: []Field{{Type: "[]string"}}}, Emit: true},
		}}
		out := imp.queryImports("tags.sql")
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/lib/pq" {
				found = true
			}
		}
		if !found {
			t.Error("expected lib/pq for slice array with database/sql")
		}
	})

	t.Run("emit_err_nil_if_no_rows_pgxv5_adds_errors_and_pgx", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Options.EmitErrNilIfNoRows = true
		imp.Queries = []Query{{Cmd: metadata.CmdOne, SourceName: "users.sql", SQL: "SELECT id FROM users WHERE id = $1"}}
		out := imp.queryImports("users.sql")
		hasErrors, hasPgx := false, false
		for _, s := range out.Std {
			if s.Path == "errors" {
				hasErrors = true
			}
		}
		for _, s := range out.Dep {
			if s.Path == "github.com/jackc/pgx/v5" {
				hasPgx = true
			}
		}
		if !hasErrors || !hasPgx {
			t.Errorf("missing imports: errors=%v pgx=%v", hasErrors, hasPgx)
		}
	})

	t.Run("tracing_option_adds_tracing_import", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Options.EmitTracing = &opts.TracingOptions{Import: "go.opentelemetry.io/otel", Package: "otel"}
		imp.Queries = []Query{{Cmd: metadata.CmdOne, SourceName: "users.sql"}}
		out := imp.queryImports("users.sql")
		found := false
		for _, s := range out.Dep {
			if s.Path == "go.opentelemetry.io/otel" {
				found = true
			}
		}
		if !found {
			t.Error("expected otel import when EmitTracing configured")
		}
	})
}

func TestInterfaceImports(t *testing.T) {
	t.Run("always_adds_context", func(t *testing.T) {
		imp := baseImp("database/sql")
		out := imp.interfaceImports()
		found := false
		for _, s := range out.Std {
			if s.Path == "context" {
				found = true
			}
		}
		if !found {
			t.Error("expected context in interface imports")
		}
	})

	t.Run("ret_type_uuid_adds_uuid_import", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Queries = []Query{{Cmd: metadata.CmdOne, Ret: QueryValue{Typ: "uuid.UUID"}}}
		out := imp.interfaceImports()
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/google/uuid" {
				found = true
			}
		}
		if !found {
			t.Error("expected uuid import from ret type")
		}
	})

	t.Run("batch_ret_type_skipped", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{{Cmd: metadata.CmdBatchMany, Ret: QueryValue{Typ: "uuid.UUID"}}}
		out := imp.interfaceImports()
		for _, s := range out.Dep {
			if s.Path == "github.com/google/uuid" {
				t.Error("batch query ret type should be skipped in interface imports")
			}
		}
	})

	t.Run("arg_struct_fields_scanned_for_types", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Queries = []Query{{
			Cmd: metadata.CmdOne,
			Arg: QueryValue{Struct: &Struct{Fields: []Field{{Type: "uuid.UUID", Name: "UserID"}}}},
		}}
		out := imp.interfaceImports()
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/google/uuid" {
				found = true
			}
		}
		if !found {
			t.Error("expected uuid import from arg struct field")
		}
	})
}

func TestBatchImports(t *testing.T) {
	t.Run("always_has_context_and_errors", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{{Cmd: metadata.CmdBatchExec}}
		out := imp.batchImports()
		hasCtx, hasErr := false, false
		for _, s := range out.Std {
			switch s.Path {
			case "context":
				hasCtx = true
			case "errors":
				hasErr = true
			}
		}
		if !hasCtx || !hasErr {
			t.Errorf("missing: context=%v errors=%v", hasCtx, hasErr)
		}
	})

	t.Run("pgxv5_adds_pgx_v5", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{{Cmd: metadata.CmdBatchExec}}
		out := imp.batchImports()
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/jackc/pgx/v5" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgx/v5 in batch imports")
		}
	})

	t.Run("pgxv4_adds_pgx_v4", func(t *testing.T) {
		imp := baseImp("pgx/v4")
		imp.Queries = []Query{{Cmd: metadata.CmdBatchExec}}
		out := imp.batchImports()
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/jackc/pgx/v4" {
				found = true
			}
		}
		if !found {
			t.Error("expected pgx/v4 in batch imports")
		}
	})

	t.Run("ret_struct_field_scanned_for_types", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{{
			Cmd: metadata.CmdBatchOne,
			Ret: QueryValue{Struct: &Struct{Fields: []Field{{Type: "uuid.UUID"}}}, Emit: true},
		}}
		out := imp.batchImports()
		found := false
		for _, s := range out.Dep {
			if s.Path == "github.com/google/uuid" {
				found = true
			}
		}
		if !found {
			t.Error("expected uuid import from batch ret struct field")
		}
	})
}

func TestCopyfromImports(t *testing.T) {
	t.Run("always_has_context", func(t *testing.T) {
		imp := baseImp("pgx/v5")
		imp.Queries = []Query{{Cmd: metadata.CmdCopyFrom, Arg: QueryValue{Typ: "string"}}}
		out := imp.copyfromImports()
		found := false
		for _, s := range out.Std {
			if s.Path == "context" {
				found = true
			}
		}
		if !found {
			t.Error("expected context in copyfrom imports")
		}
	})

	t.Run("mysql_driver_adds_io_fmt_sync_mysql_mysqltsv", func(t *testing.T) {
		imp := baseImp("database/sql")
		imp.Options.SqlDriver = opts.SQLDriverGoSQLDriverMySQL
		imp.Queries = []Query{{Cmd: metadata.CmdCopyFrom, Arg: QueryValue{Typ: "string"}}}
		out := imp.copyfromImports()
		stdPaths := map[string]bool{}
		for _, s := range out.Std {
			stdPaths[s.Path] = true
		}
		depPaths := map[string]bool{}
		for _, s := range out.Dep {
			depPaths[s.Path] = true
		}
		for _, p := range []string{"io", "fmt", "sync/atomic"} {
			if !stdPaths[p] {
				t.Errorf("expected std import %q for MySQL copyfrom", p)
			}
		}
		for _, p := range []string{"github.com/go-sql-driver/mysql", "github.com/hexon/mysqltsv"} {
			if !depPaths[p] {
				t.Errorf("expected dep import %q for MySQL copyfrom", p)
			}
		}
	})
}

func TestHasPrefixIgnoringSliceAndPointerPrefix(t *testing.T) {
	cases := []struct {
		s, prefix string
		want      bool
	}{
		{"time.Time", "time.", true},
		{"[]time.Time", "time.", true},
		{"*time.Time", "time.", true},
		{"string", "time.", false},
		{"sql.NullString", "sql.", true},
	}
	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			if got := hasPrefixIgnoringSliceAndPointerPrefix(tc.s, tc.prefix); got != tc.want {
				t.Errorf("hasPrefixIgnoringSliceAndPointerPrefix(%q, %q) = %v, want %v", tc.s, tc.prefix, got, tc.want)
			}
		})
	}
}

func TestTrimSliceAndPointerPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"[]string", "string"},
		{"*string", "string"},
		{"string", "string"},
		{"[]*string", "string"}, // trims [] then *
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := trimSliceAndPointerPrefix(tc.in); got != tc.want {
				t.Errorf("trimSliceAndPointerPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
