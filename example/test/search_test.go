package db

import (
	"example/db"
	"strings"
	"testing"
)

func strPtr(v string) *string { return &v }

// DynamicSQL removes annotated lines but returns the original args unchanged.
// PostgreSQL accepts non-sequential $N (e.g. $1, $3 with no $2).

func TestDynamicSQL_SearchUsers_AllFiltersNil(t *testing.T) {
	var email, phone *string
	sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})

	if strings.Contains(sql, "AND email") {
		t.Errorf("email condition should be removed when Email is nil, got:\n%s", sql)
	}
	if strings.Contains(sql, "AND phone") {
		t.Errorf("phone condition should be removed when Phone is nil, got:\n%s", sql)
	}
	if !strings.Contains(sql, "WHERE") {
		t.Errorf("WHERE clause should remain, got:\n%s", sql)
	}
	// Original args returned unchanged
	if len(args) != 3 {
		t.Errorf("expected 3 args (original slice), got %d", len(args))
	}
	if args[0] != "alice" {
		t.Errorf("expected args[0] = \"alice\", got %v", args[0])
	}
	t.Logf("SQL:\n%s", sql)
}

func TestDynamicSQL_SearchUsers_EmailOnly(t *testing.T) {
	email := strPtr("alice@example.com")
	var phone *string

	sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})

	if !strings.Contains(sql, "AND email") {
		t.Errorf("expected email condition to be present, got:\n%s", sql)
	}
	if strings.Contains(sql, "AND phone") {
		t.Errorf("phone condition should be removed when Phone is nil, got:\n%s", sql)
	}
	if !strings.Contains(sql, "$2") {
		t.Errorf("expected $2 in SQL, got:\n%s", sql)
	}
	if strings.Contains(sql, "$3") {
		t.Errorf("expected no $3 after removal, got:\n%s", sql)
	}
	// Original args returned unchanged
	if len(args) != 3 {
		t.Errorf("expected 3 args (original slice), got %d", len(args))
	}
	if args[1] != email {
		t.Errorf("expected args[1] = email pointer, got %v", args[1])
	}
	t.Logf("SQL:\n%s", sql)
}

func TestDynamicSQL_SearchUsers_PhoneOnly(t *testing.T) {
	var email *string
	phone := strPtr("+1234567890")

	sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})

	if strings.Contains(sql, "AND email") {
		t.Errorf("email condition should be removed when Email is nil, got:\n%s", sql)
	}
	if !strings.Contains(sql, "AND phone") {
		t.Errorf("expected phone condition to be present, got:\n%s", sql)
	}
	// $3 kept as-is (no renumbering)
	if !strings.Contains(sql, "$3") {
		t.Errorf("expected $3 in SQL (original placeholder), got:\n%s", sql)
	}
	if strings.Contains(sql, "$2") {
		t.Errorf("expected no $2 after email removal, got:\n%s", sql)
	}
	// Original args returned unchanged
	if len(args) != 3 {
		t.Errorf("expected 3 args (original slice), got %d", len(args))
	}
	if args[2] != phone {
		t.Errorf("expected args[2] = phone pointer, got %v", args[2])
	}
	t.Logf("SQL:\n%s", sql)
}

func TestDynamicSQL_SearchUsers_AllFilters(t *testing.T) {
	email := strPtr("alice@example.com")
	phone := strPtr("+1234567890")

	sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})

	if !strings.Contains(sql, "AND email") {
		t.Errorf("expected email condition, got:\n%s", sql)
	}
	if !strings.Contains(sql, "AND phone") {
		t.Errorf("expected phone condition, got:\n%s", sql)
	}
	if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") || !strings.Contains(sql, "$3") {
		t.Errorf("expected $1, $2, $3 in SQL, got:\n%s", sql)
	}
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
	t.Logf("SQL:\n%s", sql)
}

func TestDynamicSQL_SearchUsers_OrderByPreserved(t *testing.T) {
	var email, phone *string
	sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})
	if !strings.Contains(sql, "ORDER BY id ASC") {
		t.Errorf("ORDER BY should always be present, got:\n%s", sql)
	}
}

func TestDynamicSQL_SearchUsers_SQLStructure(t *testing.T) {
	var email, phone *string
	sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})
	t.Logf("SQL with no optional filters:\n%s", sql)

	email = strPtr("x@y.com")
	sql, _ = db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})
	t.Logf("SQL with email only:\n%s", sql)
}
