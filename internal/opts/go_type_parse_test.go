package opts

import (
	"testing"
)

func TestGoTypeParse(t *testing.T) {
	t.Run("basic_go_type_from_spec", func(t *testing.T) {
		gt := GoType{Spec: "string"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !o.BasicType {
			t.Error("expected BasicType=true for 'string'")
		}
		if o.TypeName != "string" {
			t.Errorf("TypeName = %q, want string", o.TypeName)
		}
	})

	t.Run("qualified_type_from_spec", func(t *testing.T) {
		gt := GoType{Spec: "github.com/segmentio/ksuid.KSUID"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.ImportPath != "github.com/segmentio/ksuid" {
			t.Errorf("ImportPath = %q, want github.com/segmentio/ksuid", o.ImportPath)
		}
		if o.TypeName != "ksuid.KSUID" {
			t.Errorf("TypeName = %q, want ksuid.KSUID", o.TypeName)
		}
		if o.BasicType {
			t.Error("expected BasicType=false for package type")
		}
	})

	t.Run("spec_with_versioned_import", func(t *testing.T) {
		gt := GoType{Spec: "github.com/jackc/pgx/v5/pgtype.Text"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.ImportPath != "github.com/jackc/pgx/v5/pgtype" {
			t.Errorf("ImportPath = %q", o.ImportPath)
		}
		if o.TypeName != "pgtype.Text" {
			t.Errorf("TypeName = %q, want pgtype.Text", o.TypeName)
		}
	})

	t.Run("spec_pointer_type", func(t *testing.T) {
		gt := GoType{Spec: "*github.com/segmentio/ksuid.KSUID"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.TypeName != "*ksuid.KSUID" {
			t.Errorf("TypeName = %q, want *ksuid.KSUID", o.TypeName)
		}
		if o.ImportPath != "github.com/segmentio/ksuid" {
			t.Errorf("ImportPath = %q", o.ImportPath)
		}
	})

	t.Run("spec_invalid_basic_type", func(t *testing.T) {
		gt := GoType{Spec: "NotAGoBasicType"}
		_, err := gt.parse()
		if err == nil {
			t.Error("expected error for unknown basic type")
		}
	})

	t.Run("spec_slash_no_dot_returns_error", func(t *testing.T) {
		// A spec with a slash but no dot cannot be a valid package path
		gt := GoType{Spec: "gopkg/notype"}
		_, err := gt.parse()
		if err == nil {
			t.Error("expected error when spec has slash but no dot")
		}
	})

	t.Run("struct_fields_path_and_name", func(t *testing.T) {
		gt := GoType{Path: "time", Name: "Time"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.TypeName != "time.Time" {
			t.Errorf("TypeName = %q, want time.Time", o.TypeName)
		}
		if o.ImportPath != "time" {
			t.Errorf("ImportPath = %q, want time", o.ImportPath)
		}
	})

	t.Run("struct_fields_pointer", func(t *testing.T) {
		gt := GoType{Path: "time", Name: "Time", Pointer: true}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.TypeName != "*time.Time" {
			t.Errorf("TypeName = %q, want *time.Time", o.TypeName)
		}
	})

	t.Run("struct_fields_slice", func(t *testing.T) {
		gt := GoType{Path: "time", Name: "Time", Slice: true}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.TypeName != "[]time.Time" {
			t.Errorf("TypeName = %q, want []time.Time", o.TypeName)
		}
	})

	t.Run("struct_fields_package_without_path_errors", func(t *testing.T) {
		gt := GoType{Package: "mypkg", Name: "MyType"}
		_, err := gt.parse()
		if err == nil {
			t.Error("expected error when Package set but Path empty")
		}
	})

	t.Run("struct_fields_explicit_package_alias", func(t *testing.T) {
		gt := GoType{Path: "github.com/foo/bar", Package: "baz", Name: "MyType"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.Package != "baz" {
			t.Errorf("Package = %q, want baz", o.Package)
		}
		if o.TypeName != "baz.MyType" {
			t.Errorf("TypeName = %q, want baz.MyType", o.TypeName)
		}
	})

	t.Run("struct_fields_basic_type_no_path", func(t *testing.T) {
		gt := GoType{Name: "string"}
		o, err := gt.parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !o.BasicType {
			t.Error("expected BasicType=true when no path/package")
		}
		if o.TypeName != "string" {
			t.Errorf("TypeName = %q, want string", o.TypeName)
		}
	})
}

func TestGoStructTagParse(t *testing.T) {
	t.Run("empty_tag", func(t *testing.T) {
		m, err := GoStructTag("").parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %v", m)
		}
	})

	t.Run("single_tag", func(t *testing.T) {
		m, err := GoStructTag(`validate:"required"`).parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["validate"] != "required" {
			t.Errorf("validate = %q, want required", m["validate"])
		}
	})

	t.Run("multiple_tags", func(t *testing.T) {
		m, err := GoStructTag(`validate:"required" form:"name"`).parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["validate"] != "required" || m["form"] != "name" {
			t.Errorf("unexpected tags: %v", m)
		}
	})

	t.Run("invalid_tag_returns_error", func(t *testing.T) {
		_, err := GoStructTag(`notvalid`).parse()
		if err == nil {
			t.Error("expected error for invalid struct tag")
		}
	})
}
