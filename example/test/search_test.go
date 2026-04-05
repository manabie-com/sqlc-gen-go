package db

import (
	"example/db"
	"strings"
	"testing"
	"time"
)

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }

func assertContains(t *testing.T, sql, substr string) {
	t.Helper()
	if !strings.Contains(sql, substr) {
		t.Errorf("expected %q in:\n%s", substr, sql)
	}
}

func assertAbsent(t *testing.T, sql, substr string) {
	t.Helper()
	if strings.Contains(sql, substr) {
		t.Errorf("unexpected %q in:\n%s", substr, sql)
	}
}

func TestSearchUsers(t *testing.T) {
	now := time.Now()

	// $1=Name, $2=Email, $3=Phone, $4=OrdersSince, $5=HasOrders
	args := func(email *string, phone *string, ordersSince *time.Time, hasOrders bool) []any {
		return []any{"alice", email, phone, ordersSince, hasOrders}
	}

	t.Run("NoOptionalFilters", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, nil, nil, false))
		assertAbsent(t, sql, "AND email")
		assertAbsent(t, sql, "AND phone")
		assertAbsent(t, sql, "AND EXISTS")
		assertContains(t, sql, "WHERE")
		assertContains(t, sql, "ORDER BY id ASC")
		if len(a) != 1 {
			t.Errorf("expected 1 arg (only name/$1), got %d", len(a))
		}
	})

	t.Run("EmailOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(strPtr("alice@example.com"), nil, nil, false))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "$2")
		assertAbsent(t, sql, "AND phone")
		assertAbsent(t, sql, "$3")
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(nil, strPtr("+1234567890"), nil, false))
		assertAbsent(t, sql, "AND email")
		assertContains(t, sql, "AND phone")
		// email ($2) removed → phone renumbered from $3 to $2
		assertContains(t, sql, "$2")
		assertAbsent(t, sql, "$3")
	})

	t.Run("EmailAndPhone", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(strPtr("a@b.com"), strPtr("+1"), nil, false))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "AND phone")
	})

	t.Run("HasOrders_False", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(nil, nil, nil, false))
		assertAbsent(t, sql, "AND EXISTS")
		assertAbsent(t, sql, "orders.user_id")
	})

	t.Run("HasOrders_True_NoOrdersSince", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(nil, nil, nil, true))
		assertContains(t, sql, "AND EXISTS")
		assertContains(t, sql, "orders.user_id = users.id")
		assertAbsent(t, sql, "orders.created_at")
	})

	t.Run("HasOrders_True_WithOrdersSince", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, args(nil, nil, &now, true))
		assertContains(t, sql, "AND EXISTS")
		// email($2) and phone($3) removed → ordersSince renumbered from $4 to $2
		assertContains(t, sql, "orders.created_at >= $2")
	})

	t.Run("HasOrders_False_OrdersSince_Ignored", func(t *testing.T) {
		// OrdersSince is set but HasOrders=false — the whole EXISTS block should be dropped
		sql, _ := db.DynamicSQL(db.SearchUsers, args(nil, nil, &now, false))
		assertAbsent(t, sql, "AND EXISTS")
		assertAbsent(t, sql, "orders.created_at")
	})

	t.Run("AllFilters", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(
			strPtr("alice@example.com"), strPtr("+1234567890"), &now, true,
		))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "AND phone")
		assertContains(t, sql, "AND EXISTS")
		// All SQL params kept: $1=name, $2=email, $3=phone, $4=ordersSince
		// $5=hasOrders is annotation-only (bool flag), not a SQL placeholder
		assertContains(t, sql, "orders.created_at >= $4")
		if len(a) != 4 {
			t.Errorf("expected 4 args ($1-$4), got %d", len(a))
		}
	})
}

func TestSearchUsersOrdered(t *testing.T) {
	t.Run("NoOrderFlags", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, false, false})
		assertAbsent(t, sql, "created_at DESC")
		assertAbsent(t, sql, "name ASC,")
		assertContains(t, sql, "id ASC")
	})

	t.Run("CreatedAtDesc", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, true, false})
		assertContains(t, sql, "created_at DESC,")
		assertAbsent(t, sql, "name ASC,")
		assertContains(t, sql, "id ASC")
	})

	t.Run("NameAsc", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, false, true})
		assertAbsent(t, sql, "created_at DESC")
		assertContains(t, sql, "name ASC,")
		assertContains(t, sql, "id ASC")
	})

	t.Run("AllFlags", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", strPtr("alice@example.com"), true, true})
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "created_at DESC,")
		assertContains(t, sql, "name ASC,")
		assertContains(t, sql, "id ASC")
		// $3 and $4 are bool-flag annotations only, not SQL placeholders → only $1, $2 returned
		if len(a) != 2 {
			t.Errorf("expected 2 args ($1=name, $2=email), got %d", len(a))
		}
	})
}

func TestSearchUsersByContact(t *testing.T) {
	t.Run("BothNil", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", nil, nil})
		assertAbsent(t, sql, "email = $2 OR phone")
	})

	t.Run("EmailOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", strPtr("alice@example.com"), nil})
		assertAbsent(t, sql, "email = $2 OR phone")
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", nil, strPtr("+1234567890")})
		assertAbsent(t, sql, "email = $2 OR phone")
	})

	t.Run("Both", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", strPtr("alice@example.com"), strPtr("+1234567890")})
		assertContains(t, sql, "email = $2 OR phone = $3")
		if len(a) != 3 {
			t.Errorf("expected 3 args, got %d", len(a))
		}
	})
}

func TestSearchUsersOrderedByID(t *testing.T) {
	t.Run("IDAsc", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, boolPtr(true), nil})
		assertContains(t, sql, "ASC")
		assertAbsent(t, sql, "DESC")
		assertAbsent(t, sql, "ASC,") // trailing comma must be stripped
		// $1=name, $2=IDAsc (bool pointer kept as SQL param)
		if len(a) != 2 {
			t.Errorf("args len: got %d, want 2", len(a))
		}
	})

	t.Run("IDDesc", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, nil, boolPtr(true)})
		assertAbsent(t, sql, "ASC")
		assertContains(t, sql, "DESC")
		// $1=name, $2=IDDesc (renumbered from $4)
		if len(a) != 2 {
			t.Errorf("args len: got %d, want 2", len(a))
		}
	})

	t.Run("BothFlags", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, boolPtr(true), boolPtr(true)})
		assertContains(t, sql, "ASC")
		assertContains(t, sql, "DESC")
		// $1=name, $2=IDAsc, $3=IDDesc
		if len(a) != 3 {
			t.Errorf("args len: got %d, want 3", len(a))
		}
	})

	t.Run("WithEmail", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", strPtr("alice@example.com"), boolPtr(true), boolPtr(true)})
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "ASC")
		assertContains(t, sql, "DESC")
		// $1=name, $2=email, $3=IDAsc, $4=IDDesc
		if len(a) != 4 {
			t.Errorf("args len: got %d, want 4", len(a))
		}
	})
}
