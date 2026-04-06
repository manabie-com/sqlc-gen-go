package opts

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/pattern"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func mustMatch(t *testing.T, expr string) *pattern.Match {
	t.Helper()
	m, err := pattern.MatchCompile(expr)
	if err != nil {
		t.Fatalf("MatchCompile(%q): %v", expr, err)
	}
	return m
}

func TestOverride_Matches_NilIdentifier(t *testing.T) {
	o := &Override{}
	if o.Matches(nil, "public") {
		t.Error("expected false for nil identifier")
	}
}

func TestOverride_Matches_NoConstraints(t *testing.T) {
	// No TableRel/Schema set, identifier with empty name -> matches
	o := &Override{}
	n := &plugin.Identifier{Name: ""}
	if !o.Matches(n, "") {
		t.Error("expected true when no constraints and empty identifier")
	}
}

func TestOverride_Matches_TableRelMatch(t *testing.T) {
	o := &Override{
		TableRel:    mustMatch(t, "users"),
		TableSchema: mustMatch(t, "public"),
	}
	n := &plugin.Identifier{Schema: "public", Name: "users"}
	if !o.Matches(n, "public") {
		t.Error("expected true for matching table rel and schema")
	}
}

func TestOverride_Matches_TableRelMismatch(t *testing.T) {
	o := &Override{
		TableRel:    mustMatch(t, "orders"),
		TableSchema: mustMatch(t, "public"),
	}
	n := &plugin.Identifier{Schema: "public", Name: "users"}
	if o.Matches(n, "public") {
		t.Error("expected false when table rel doesn't match")
	}
}

func TestOverride_Matches_SchemaMismatch(t *testing.T) {
	o := &Override{
		TableRel:    mustMatch(t, "users"),
		TableSchema: mustMatch(t, "other"),
	}
	n := &plugin.Identifier{Name: "users"}
	if o.Matches(n, "public") {
		t.Error("expected false when schema doesn't match")
	}
}

func TestOverride_Matches_TableRelNilWithName(t *testing.T) {
	// TableRel is nil but identifier has a name -> false
	o := &Override{}
	n := &plugin.Identifier{Name: "users"}
	if o.Matches(n, "") {
		t.Error("expected false when TableRel is nil but identifier has name")
	}
}

func TestOverride_Matches_SchemaFromDefault(t *testing.T) {
	o := &Override{
		TableRel:    mustMatch(t, "users"),
		TableSchema: mustMatch(t, "myschema"),
	}
	// n.Schema is empty, defaultSchema is "myschema"
	n := &plugin.Identifier{Name: "users"}
	if !o.Matches(n, "myschema") {
		t.Error("expected true when schema falls back to default")
	}
}

func minReq() *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{DefaultSchema: "public"},
	}
}

func TestOverride_Parse_ColumnFormats(t *testing.T) {
	cases := []struct {
		col     string
		wantErr bool
	}{
		{"users.id", false},
		{"public.users.id", false},
		{"mydb.public.users.id", false},
		{"id", true},        // too few parts
		{"a.b.c.d.e", true}, // too many parts
	}
	for _, tc := range cases {
		t.Run(tc.col, func(t *testing.T) {
			o := &Override{
				Column: tc.col,
				GoType: GoType{Spec: "string"},
			}
			err := o.parse(minReq())
			if tc.wantErr && err == nil {
				t.Errorf("expected error for column %q", tc.col)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for column %q: %v", tc.col, err)
			}
		})
	}
}

func TestOverride_Parse_BothColumnAndDBType(t *testing.T) {
	o := &Override{
		Column: "users.id",
		DBType: "uuid",
		GoType: GoType{Spec: "string"},
	}
	if err := o.parse(nil); err == nil {
		t.Error("expected error when both column and db_type are set")
	}
}

func TestOverride_Parse_NeitherColumnNorDBType(t *testing.T) {
	o := &Override{GoType: GoType{Spec: "string"}}
	if err := o.parse(nil); err == nil {
		t.Error("expected error when neither column nor db_type is set")
	}
}

func TestOverride_Parse_DeprecatedPostgresType(t *testing.T) {
	o := &Override{
		Deprecated_PostgresType: "uuid",
		GoType:                  GoType{Spec: "string"},
	}
	if err := o.parse(minReq()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if o.DBType != "uuid" {
		t.Errorf("DBType = %q, want uuid", o.DBType)
	}
}

func TestOverride_Parse_DeprecatedNull(t *testing.T) {
	o := &Override{
		DBType:          "uuid",
		Deprecated_Null: true,
		GoType:          GoType{Spec: "string"},
	}
	if err := o.parse(minReq()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !o.Nullable {
		t.Error("expected Nullable=true after deprecated null migration")
	}
}
