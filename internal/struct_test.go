package golang

import (
	"testing"

	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func defaultOpts() *opts.Options {
	return &opts.Options{
		InitialismsMap: map[string]struct{}{"id": {}},
		Rename:         map[string]string{},
	}
}

func TestStructName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"users", "Users"},
		{"user_account", "UserAccount"},
		{"user_id", "UserID"},
		{"order_items", "OrderItems"},
		// digits at start get underscore prefix
		{"123table", "_123table"},
		// non-letter non-digit chars become underscore
		{"my-table", "MyTable"},
	}
	opts := defaultOpts()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StructName(tc.name, opts)
			if got != tc.want {
				t.Errorf("StructName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestStructName_Rename(t *testing.T) {
	o := &opts.Options{
		InitialismsMap: map[string]struct{}{},
		Rename:         map[string]string{"users": "AppUser"},
	}
	got := StructName("users", o)
	if got != "AppUser" {
		t.Errorf("StructName with rename = %q, want %q", got, "AppUser")
	}
}
