package db

import (
	"example/db"
	"strings"
	"testing"
	"time"
)

func strPtr(v string) *string { return &v }

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

// buildSearchUsersArgs builds the 5-element args slice for SearchUsers:
// $1=Name, $2=Email, $3=Phone, $4=OrdersSince, $5=HasOrders
func buildSearchUsersArgs(email *string, phone *string, ordersSince *time.Time, hasOrders bool) []any {
	return []any{"alice", email, phone, ordersSince, hasOrders}
}

func TestDynamicSQL_SearchUsers(t *testing.T) {
	now := time.Now()

	t.Run("NoOptionalFilters", func(t *testing.T) {
		sql, args := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, nil, nil, false))
		assertAbsent(t, sql, "AND email")
		assertAbsent(t, sql, "AND phone")
		assertAbsent(t, sql, "AND EXISTS")
		assertContains(t, sql, "WHERE")
		assertContains(t, sql, "ORDER BY id ASC")
		if len(args) != 5 {
			t.Errorf("expected 5 args (original slice), got %d", len(args))
		}
	})

	t.Run("EmailOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(strPtr("alice@example.com"), nil, nil, false))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "$2")
		assertAbsent(t, sql, "AND phone")
		assertAbsent(t, sql, "$3")
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, strPtr("+1234567890"), nil, false))
		assertAbsent(t, sql, "AND email")
		assertAbsent(t, sql, "$2")
		assertContains(t, sql, "AND phone")
		assertContains(t, sql, "$3")
	})

	t.Run("EmailAndPhone", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(strPtr("a@b.com"), strPtr("+1"), nil, false))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "AND phone")
	})

	t.Run("HasOrders_False", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, nil, nil, false))
		assertAbsent(t, sql, "AND EXISTS")
		assertAbsent(t, sql, "orders.user_id")
	})

	t.Run("HasOrders_True_NoOrdersSince", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, nil, nil, true))
		assertContains(t, sql, "AND EXISTS")
		assertContains(t, sql, "orders.user_id = users.id")
		assertAbsent(t, sql, "orders.created_at")
	})

	t.Run("HasOrders_True_WithOrdersSince", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, nil, &now, true))
		assertContains(t, sql, "AND EXISTS")
		assertContains(t, sql, "orders.created_at >= $4")
	})

	t.Run("HasOrders_False_OrdersSince_Ignored", func(t *testing.T) {
		// OrdersSince is set but HasOrders=false — the whole EXISTS block should be dropped
		sql, _ := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(nil, nil, &now, false))
		assertAbsent(t, sql, "AND EXISTS")
		assertAbsent(t, sql, "orders.created_at")
	})

	t.Run("AllFilters", func(t *testing.T) {
		sql, args := db.DynamicSQL(db.SearchUsers, buildSearchUsersArgs(
			strPtr("alice@example.com"), strPtr("+1234567890"), &now, true,
		))
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "AND phone")
		assertContains(t, sql, "AND EXISTS")
		assertContains(t, sql, "orders.created_at >= $4")
		if len(args) != 5 {
			t.Errorf("expected 5 args, got %d", len(args))
		}
	})
}

// SearchUsersOrdered: dynamic ORDER BY with flag-only bool params
func TestDynamicSQL_SearchUsersOrdered(t *testing.T) {
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
		sql, args := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", strPtr("alice@example.com"), true, true})
		assertContains(t, sql, "AND email")
		assertContains(t, sql, "created_at DESC,")
		assertContains(t, sql, "name ASC,")
		assertContains(t, sql, "id ASC")
		if len(args) != 4 {
			t.Errorf("expected 4 args, got %d", len(args))
		}
	})
}

// SearchUsersByContact: multi-param :if — line dropped if ANY param is nil
func TestDynamicSQL_SearchUsersByContact(t *testing.T) {
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
		sql, args := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", strPtr("alice@example.com"), strPtr("+1234567890")})
		assertContains(t, sql, "email = $2 OR phone = $3")
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
	})
}
