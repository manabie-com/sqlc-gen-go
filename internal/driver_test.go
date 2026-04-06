package golang

import "testing"

func TestParseDriver_All(t *testing.T) {
	cases := []struct {
		pkg  string
		want string
	}{
		{"pgx/v4", "github.com/jackc/pgx/v4"},
		{"pgx/v5", "github.com/jackc/pgx/v5"},
		{"database/sql", "github.com/lib/pq"},
		{"", "github.com/lib/pq"},
	}
	for _, tc := range cases {
		got := parseDriver(tc.pkg)
		if string(got) != tc.want {
			t.Errorf("parseDriver(%q) = %q, want %q", tc.pkg, got, tc.want)
		}
	}
}
