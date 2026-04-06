package golang

import "testing"

func TestIsReserved(t *testing.T) {
	reserved := []string{
		"break", "default", "func", "interface", "select",
		"case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var",
	}
	for _, kw := range reserved {
		if !IsReserved(kw) {
			t.Errorf("IsReserved(%q) = false, want true", kw)
		}
	}

	notReserved := []string{"name", "user", "id", "query", "result"}
	for _, w := range notReserved {
		if IsReserved(w) {
			t.Errorf("IsReserved(%q) = true, want false", w)
		}
	}
}

func TestEscape(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"name", "name"},
		{"select", "select_"},
		{"user", "user"},
		{"func", "func_"},
	}
	for _, tc := range cases {
		got := escape(tc.in)
		if got != tc.want {
			t.Errorf("escape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
