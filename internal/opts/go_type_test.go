package opts

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGoType_MarshalJSON_Spec(t *testing.T) {
	gt := GoType{Spec: "string"}
	b, err := gt.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != `"string"` {
		t.Errorf("MarshalJSON = %s, want %q", b, `"string"`)
	}
}

func TestGoType_MarshalJSON_Struct(t *testing.T) {
	gt := GoType{Path: "time", Name: "Time"}
	b, err := gt.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should marshal as object
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("result is not JSON object: %v", err)
	}
}

func TestGoType_UnmarshalJSON_String(t *testing.T) {
	var gt GoType
	if err := json.Unmarshal([]byte(`"string"`), &gt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gt.Spec != "string" {
		t.Errorf("Spec = %q, want %q", gt.Spec, "string")
	}
}

func TestGoType_UnmarshalJSON_Object(t *testing.T) {
	data := `{"import":"time","type":"Time"}`
	var gt GoType
	if err := json.Unmarshal([]byte(data), &gt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gt.Path != "time" {
		t.Errorf("Path = %q, want %q", gt.Path, "time")
	}
	if gt.Name != "Time" {
		t.Errorf("Name = %q, want %q", gt.Name, "Time")
	}
}

func TestGoType_UnmarshalJSON_Invalid(t *testing.T) {
	var gt GoType
	err := json.Unmarshal([]byte(`{invalid}`), &gt)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGoType_UnmarshalYAML_String(t *testing.T) {
	var gt GoType
	err := gt.UnmarshalYAML(func(v interface{}) error {
		if sp, ok := v.(*string); ok {
			*sp = "time.Time"
			return nil
		}
		return fmt.Errorf("not a string")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gt.Spec != "time.Time" {
		t.Errorf("Spec = %q, want time.Time", gt.Spec)
	}
}

func TestGoType_UnmarshalYAML_Struct(t *testing.T) {
	// Fails string path, falls back to struct unmarshal (succeeds with zero-value alias)
	var gt GoType
	err := gt.UnmarshalYAML(func(v interface{}) error {
		if _, ok := v.(*string); ok {
			return fmt.Errorf("not a string")
		}
		return nil // succeed with whatever struct is passed
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratePackageID(t *testing.T) {
	cases := []struct {
		importPath string
		wantPkg    string
		wantAlias  bool
	}{
		{"github.com/segmentio/ksuid", "ksuid", false},
		{"github.com/jackc/pgx/v5", "pgx", true},
		{"github.com/jackc/pgx/v4", "pgx", true},
		{"time", "time", false},
		{"github.com/go-sql-driver/mysql", "mysql", false},
	}
	for _, tc := range cases {
		t.Run(tc.importPath, func(t *testing.T) {
			pkg, alias := generatePackageID(tc.importPath)
			if pkg != tc.wantPkg {
				t.Errorf("generatePackageID(%q) pkg = %q, want %q", tc.importPath, pkg, tc.wantPkg)
			}
			if alias != tc.wantAlias {
				t.Errorf("generatePackageID(%q) alias = %v, want %v", tc.importPath, alias, tc.wantAlias)
			}
		})
	}
}
