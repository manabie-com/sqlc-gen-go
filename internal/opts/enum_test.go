package opts

import "testing"

func TestValidatePackage(t *testing.T) {
	valid := []string{"pgx/v4", "pgx/v5", "database/sql"}
	for _, pkg := range valid {
		if err := validatePackage(pkg); err != nil {
			t.Errorf("validatePackage(%q) unexpected error: %v", pkg, err)
		}
	}
	if err := validatePackage("unknown"); err == nil {
		t.Error("validatePackage(unknown) expected error")
	}
}

func TestValidateDriver(t *testing.T) {
	valid := []string{
		"github.com/jackc/pgx/v4",
		"github.com/jackc/pgx/v5",
		"github.com/lib/pq",
		"github.com/go-sql-driver/mysql",
	}
	for _, drv := range valid {
		if err := validateDriver(drv); err != nil {
			t.Errorf("validateDriver(%q) unexpected error: %v", drv, err)
		}
	}
	if err := validateDriver("unknown"); err == nil {
		t.Error("validateDriver(unknown) expected error")
	}
}

func TestSQLDriverIsPGX(t *testing.T) {
	cases := []struct {
		drv  SQLDriver
		want bool
	}{
		{SQLDriverPGXV4, true},
		{SQLDriverPGXV5, true},
		{SQLDriverLibPQ, false},
		{SQLDriverGoSQLDriverMySQL, false},
	}
	for _, tc := range cases {
		got := tc.drv.IsPGX()
		if got != tc.want {
			t.Errorf("IsPGX(%q) = %v, want %v", tc.drv, got, tc.want)
		}
	}
}

func TestSQLDriverIsGoSQLDriverMySQL(t *testing.T) {
	if !SQLDriver(SQLDriverGoSQLDriverMySQL).IsGoSQLDriverMySQL() {
		t.Error("expected IsGoSQLDriverMySQL() = true")
	}
	if SQLDriver(SQLDriverPGXV5).IsGoSQLDriverMySQL() {
		t.Error("expected IsGoSQLDriverMySQL() = false for pgx/v5")
	}
}

func TestSQLDriverPackage(t *testing.T) {
	cases := []struct {
		drv  SQLDriver
		want string
	}{
		{SQLDriverPGXV4, SQLPackagePGXV4},
		{SQLDriverPGXV5, SQLPackagePGXV5},
		{SQLDriverLibPQ, SQLPackageStandard},
		{SQLDriverGoSQLDriverMySQL, SQLPackageStandard},
	}
	for _, tc := range cases {
		got := tc.drv.Package()
		if got != tc.want {
			t.Errorf("Package(%q) = %q, want %q", tc.drv, got, tc.want)
		}
	}
}
