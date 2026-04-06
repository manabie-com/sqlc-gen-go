package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func mysqlCol(typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Type:    &plugin.Identifier{Name: typeName},
		NotNull: notNull,
	}
}

func mysqlColUnsigned(typeName string, notNull, unsigned bool) *plugin.Column {
	return &plugin.Column{
		Type:     &plugin.Identifier{Name: typeName},
		NotNull:  notNull,
		Unsigned: unsigned,
	}
}

func mysqlReq() *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas:       []*plugin.Schema{},
		},
	}
}

func TestMysqlType_NotNull(t *testing.T) {
	req := mysqlReq()
	o := &opts.Options{}

	cases := []struct {
		typ  string
		want string
	}{
		{"varchar", "string"},
		{"text", "string"},
		{"char", "string"},
		{"tinytext", "string"},
		{"mediumtext", "string"},
		{"longtext", "string"},
		{"year", "int16"},
		{"smallint", "int16"},
		{"int", "int32"},
		{"integer", "int32"},
		{"mediumint", "int32"},
		{"bigint", "int64"},
		{"blob", "[]byte"},
		{"binary", "[]byte"},
		{"varbinary", "[]byte"},
		{"double", "float64"},
		{"float", "float64"},
		{"real", "float64"},
		{"decimal", "string"},
		{"enum", "string"},
		{"boolean", "bool"},
		{"bool", "bool"},
		{"json", "json.RawMessage"},
		{"any", "interface{}"},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			got := mysqlType(req, o, mysqlCol(tc.typ, true))
			if got != tc.want {
				t.Errorf("mysqlType(%q, notNull=true) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestMysqlType_Nullable(t *testing.T) {
	req := mysqlReq()
	o := &opts.Options{}

	cases := []struct {
		typ  string
		want string
	}{
		{"varchar", "sql.NullString"},
		{"text", "sql.NullString"},
		{"year", "sql.NullInt16"},
		{"smallint", "sql.NullInt16"},
		{"int", "sql.NullInt32"},
		{"bigint", "sql.NullInt64"},
		{"double", "sql.NullFloat64"},
		{"decimal", "sql.NullString"},
		{"boolean", "sql.NullBool"},
		{"date", "sql.NullTime"},
		{"timestamp", "sql.NullTime"},
		{"datetime", "sql.NullTime"},
		{"time", "sql.NullTime"},
	}
	for _, tc := range cases {
		t.Run(tc.typ+"_nullable", func(t *testing.T) {
			got := mysqlType(req, o, mysqlCol(tc.typ, false))
			if got != tc.want {
				t.Errorf("mysqlType(%q, notNull=false) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestMysqlType_Unsigned(t *testing.T) {
	req := mysqlReq()
	o := &opts.Options{}

	cases := []struct {
		typ  string
		want string
	}{
		{"smallint", "uint16"},
		{"int", "uint32"},
		{"bigint", "uint64"},
	}
	for _, tc := range cases {
		t.Run(tc.typ+"_unsigned", func(t *testing.T) {
			got := mysqlType(req, o, mysqlColUnsigned(tc.typ, true, true))
			if got != tc.want {
				t.Errorf("mysqlType(%q, unsigned=true) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestMysqlType_TinyintBool(t *testing.T) {
	req := mysqlReq()
	o := &opts.Options{}

	col := &plugin.Column{
		Type:    &plugin.Identifier{Name: "tinyint"},
		NotNull: true,
		Length:  1,
	}
	got := mysqlType(req, o, col)
	if got != "bool" {
		t.Errorf("tinyint(1) notNull = %q, want bool", got)
	}
}

func TestMysqlType_EnumLookup(t *testing.T) {
	req := &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{
				{
					Name: "public",
					Enums: []*plugin.Enum{
						{Name: "status", Vals: []string{"active", "inactive"}},
					},
				},
			},
		},
	}
	o := &opts.Options{
		InitialismsMap: map[string]struct{}{},
		Rename:         map[string]string{},
	}

	// notNull - in default schema returns StructName
	col := mysqlCol("status", true)
	got := mysqlType(req, o, col)
	if got != "Status" {
		t.Errorf("mysqlType(enum, notNull) = %q, want Status", got)
	}

	// nullable - returns "Null" prefix
	col2 := mysqlCol("status", false)
	got2 := mysqlType(req, o, col2)
	if got2 != "NullStatus" {
		t.Errorf("mysqlType(enum, nullable) = %q, want NullStatus", got2)
	}
}

func TestMysqlType_UnknownType(t *testing.T) {
	req := mysqlReq()
	o := &opts.Options{InitialismsMap: map[string]struct{}{}}
	got := mysqlType(req, o, mysqlCol("totally_unknown_xyz", true))
	if got != "interface{}" {
		t.Errorf("unknown type = %q, want interface{}", got)
	}
}
