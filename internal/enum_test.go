package golang

import "testing"

func TestNameTag(t *testing.T) {
	e := Enum{NameTags: map[string]string{"json": "my_enum"}}
	if got := e.NameTag(); got != `json:"my_enum"` {
		t.Errorf("NameTag() = %q, want %q", got, `json:"my_enum"`)
	}
}

func TestValidTag(t *testing.T) {
	e := Enum{ValidTags: map[string]string{"validate": "required"}}
	if got := e.ValidTag(); got != `validate:"required"` {
		t.Errorf("ValidTag() = %q, want %q", got, `validate:"required"`)
	}
}

func TestNameTagEmpty(t *testing.T) {
	e := Enum{}
	if got := e.NameTag(); got != "" {
		t.Errorf("NameTag() = %q, want empty", got)
	}
}

func TestEnumReplace(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello_world", "hello_world"},
		{"hello-world", "hello_world"},
		{"hello/world", "hello_world"},
		{"hello:world", "hello_world"},
		{"hello world", "helloworld"},
		{"abc123", "abc123"},
		{"!@#$", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := EnumReplace(tc.in)
			if got != tc.want {
				t.Errorf("EnumReplace(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEnumValueName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello_world", "HelloWorld"},
		{"foo_bar_baz", "FooBarBaz"},
		{"single", "Single"},
		{"hello-world", "HelloWorld"},
		{"UPPER", "UPPER"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := EnumValueName(tc.in)
			if got != tc.want {
				t.Errorf("EnumValueName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTitleFirst(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello", "Hello"},
		{"world", "World"},
		{"a", "A"},
		{"ABC", "ABC"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := titleFirst(tc.in)
			if got != tc.want {
				t.Errorf("titleFirst(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
