package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func TestParseIdentifierString(t *testing.T) {
	cases := []struct {
		in      string
		catalog string
		schema  string
		name    string
		wantErr bool
	}{
		{"users", "", "", "users", false},
		{"public.users", "", "public", "users", false},
		{"mydb.public.users", "mydb", "public", "users", false},
		{"a.b.c.d", "", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseIdentifierString(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Catalog != tc.catalog {
				t.Errorf("catalog = %q, want %q", got.Catalog, tc.catalog)
			}
			if got.Schema != tc.schema {
				t.Errorf("schema = %q, want %q", got.Schema, tc.schema)
			}
			if got.Name != tc.name {
				t.Errorf("name = %q, want %q", got.Name, tc.name)
			}
		})
	}
}

func makeCol(typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Type:    &plugin.Identifier{Name: typeName},
		NotNull: notNull,
	}
}

func makeReqWithCatalog() *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas:       []*plugin.Schema{},
		},
	}
}

func TestPostgresType_BasicNotNull(t *testing.T) {
	req := makeReqWithCatalog()
	o := &opts.Options{SqlPackage: "pgx/v5"}

	cases := []struct {
		typ  string
		want string
	}{
		{"serial", "int32"},
		{"bigserial", "int64"},
		{"smallserial", "int16"},
		{"integer", "int32"},
		{"int", "int32"},
		{"bigint", "int64"},
		{"smallint", "int16"},
		{"text", "string"},
		{"boolean", "bool"},
		{"json", "[]byte"},
		{"jsonb", "[]byte"},
		{"bytea", "[]byte"},
		{"any", "interface{}"},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			got := postgresType(req, o, makeCol(tc.typ, true))
			if got != tc.want {
				t.Errorf("postgresType(%q, notNull=true) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}

func TestPostgresType_NullableStdlib(t *testing.T) {
	req := makeReqWithCatalog()
	o := &opts.Options{SqlPackage: "database/sql"}

	cases := []struct {
		typ  string
		want string
	}{
		{"integer", "sql.NullInt32"},
		{"bigint", "sql.NullInt64"},
		{"smallint", "sql.NullInt16"},
		{"text", "sql.NullString"},
		{"boolean", "sql.NullBool"},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			got := postgresType(req, o, makeCol(tc.typ, false))
			if got != tc.want {
				t.Errorf("postgresType(%q, notNull=false) = %q, want %q", tc.typ, got, tc.want)
			}
		})
	}
}
