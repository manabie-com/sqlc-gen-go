package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func sqliteCol(typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Type:    &plugin.Identifier{Name: typeName},
		NotNull: notNull,
	}
}

func sqliteReq() *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "main",
			Schemas:       []*plugin.Schema{},
		},
	}
}

func TestSqliteType_NotNull(t *testing.T) {
	req := sqliteReq()
	o := &opts.Options{}

	cases := []struct {
		typ  string
		want string
	}{
		{"int", "int64"},
		{"integer", "int64"},
		{"tinyint", "int64"},
		{"smallint", "int64"},
		{"mediumint", "int64"},
		{"bigint", "int64"},
		{"blob", "[]byte"},
		{"real", "float64"},
		{"double", "float64"},
		{"float", "float64"},
		{"boolean", "bool"},
		{"bool", "bool"},
		{"date", "time.Time"},
		{"datetime", "time.Time"},
		{"timestamp", "time.Time"},
		{"any", "interface{}"},
		{"text", "string"},
		{"clob", "string"},
		{"numeric", "float64"},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			got := sqliteType(req, o, sqliteCol(tc.typ, true))
			if got != tc.want {
				t.Errorf("sqliteType(%q, notNull=true) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestSqliteType_Nullable(t *testing.T) {
	req := sqliteReq()
	o := &opts.Options{}

	cases := []struct {
		typ  string
		want string
	}{
		{"int", "sql.NullInt64"},
		{"real", "sql.NullFloat64"},
		{"boolean", "sql.NullBool"},
		{"date", "sql.NullTime"},
		{"text", "sql.NullString"},
		{"numeric", "sql.NullFloat64"},
	}
	for _, tc := range cases {
		t.Run(tc.typ+"_null", func(t *testing.T) {
			got := sqliteType(req, o, sqliteCol(tc.typ, false))
			if got != tc.want {
				t.Errorf("sqliteType(%q, notNull=false) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestSqliteType_NullablePointers(t *testing.T) {
	req := sqliteReq()
	o := &opts.Options{EmitPointersForNullTypes: true}

	cases := []struct {
		typ  string
		want string
	}{
		{"int", "*int64"},
		{"real", "*float64"},
		{"boolean", "*bool"},
		{"date", "*time.Time"},
		{"text", "*string"},
	}
	for _, tc := range cases {
		t.Run(tc.typ+"_ptr", func(t *testing.T) {
			got := sqliteType(req, o, sqliteCol(tc.typ, false))
			if got != tc.want {
				t.Errorf("sqliteType(%q, emitPointers=true) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestSqliteType_VarcharPrefix(t *testing.T) {
	req := sqliteReq()
	o := &opts.Options{}

	got := sqliteType(req, o, sqliteCol("varchar(255)", true))
	if got != "string" {
		t.Errorf("sqliteType(varchar(255)) = %q, want string", got)
	}
}

func TestSqliteType_UnknownType(t *testing.T) {
	req := sqliteReq()
	o := &opts.Options{}

	got := sqliteType(req, o, sqliteCol("unknowntype", true))
	if got != "interface{}" {
		t.Errorf("sqliteType(unknowntype) = %q, want interface{}", got)
	}
}
