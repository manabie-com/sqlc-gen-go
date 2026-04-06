package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func TestTagsToString(t *testing.T) {
	cases := []struct {
		tags map[string]string
		want string
	}{
		{nil, ""},
		{map[string]string{}, ""},
		{map[string]string{"json": "name"}, `json:"name"`},
		// two tags: alphabetical order
		{map[string]string{"db": "col", "json": "name"}, `db:"col" json:"name"`},
	}
	for _, tc := range cases {
		got := TagsToString(tc.tags)
		if got != tc.want {
			t.Errorf("TagsToString(%v) = %q, want %q", tc.tags, got, tc.want)
		}
	}
}

func TestJSONTagName(t *testing.T) {
	cases := []struct {
		name        string
		style       string
		idUppercase bool
		want        string
	}{
		{"user_id", "", false, "user_id"},
		{"user_id", "none", false, "user_id"},
		{"user_id", "snake", false, "user_id"},
		{"UserName", "snake", false, "user_name"},
		{"user_name", "camel", false, "userName"},
		{"user_id", "camel", false, "userId"},
		{"user_id", "camel", true, "userID"},
		{"user_name", "pascal", false, "UserName"},
	}
	for _, tc := range cases {
		o := &opts.Options{JsonTagsCaseStyle: tc.style, JsonTagsIdUppercase: tc.idUppercase}
		got := JSONTagName(tc.name, o)
		if got != tc.want {
			t.Errorf("JSONTagName(%q, style=%q, id_upper=%v) = %q, want %q", tc.name, tc.style, tc.idUppercase, got, tc.want)
		}
	}
}

func TestSetCaseStyle(t *testing.T) {
	cases := []struct {
		name  string
		style string
		want  string
	}{
		{"user_name", "camel", "userName"},
		{"user_name", "pascal", "UserName"},
		{"UserName", "snake", "user_name"},
	}
	for _, tc := range cases {
		got := SetCaseStyle(tc.name, tc.style)
		if got != tc.want {
			t.Errorf("SetCaseStyle(%q, %q) = %q, want %q", tc.name, tc.style, got, tc.want)
		}
	}
}

func TestSetCaseStylePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unsupported style")
		}
	}()
	SetCaseStyle("name", "kebab")
}

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"UserName", "user_name"},
		{"user_name", "user_name"},
		{"MyHTTPClient", "my_httpclient"},
		{"simple", "simple"},
	}
	for _, tc := range cases {
		got := toSnakeCase(tc.in)
		if got != tc.want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"user_name", "userName"},
		{"first_last_name", "firstLastName"},
		{"id", "id"},
	}
	for _, tc := range cases {
		got := toCamelCase(tc.in)
		if got != tc.want {
			t.Errorf("toCamelCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"user_name", "UserName"},
		{"id", "ID"},
		{"first_name", "FirstName"},
	}
	for _, tc := range cases {
		got := toPascalCase(tc.in)
		if got != tc.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToJsonCamelCase(t *testing.T) {
	cases := []struct {
		in          string
		idUppercase bool
		want        string
	}{
		{"user_name", false, "userName"},
		{"user_id", false, "userId"},
		{"user_id", true, "userID"},
		{"first_last_name", false, "firstLastName"},
	}
	for _, tc := range cases {
		got := toJsonCamelCase(tc.in, tc.idUppercase)
		if got != tc.want {
			t.Errorf("toJsonCamelCase(%q, %v) = %q, want %q", tc.in, tc.idUppercase, got, tc.want)
		}
	}
}

func TestHasSqlcSlice(t *testing.T) {
	if (Field{Column: &plugin.Column{IsSqlcSlice: true}}).HasSqlcSlice() != true {
		t.Error("expected true for IsSqlcSlice=true")
	}
	if (Field{Column: &plugin.Column{IsSqlcSlice: false}}).HasSqlcSlice() != false {
		t.Error("expected false for IsSqlcSlice=false")
	}
}

func TestSetJSONCaseStylePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unsupported style")
		}
	}()
	SetJSONCaseStyle("name", "kebab", false)
}

func TestToLowerCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"Hello", "hello"},
		{"ABC", "aBC"},
		{"already", "already"},
	}
	for _, tc := range cases {
		got := toLowerCase(tc.in)
		if got != tc.want {
			t.Errorf("toLowerCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
