package db

import (
	"example/db"
	"strings"
	"testing"
	"time"
)

func strPtr(v string) *string { return &v }

// DynamicSQL removes annotated lines but returns the original args unchanged.
// PostgreSQL accepts non-sequential $N (e.g. $1, $3 with no $2).

func TestDynamicSQL_SearchUsers(t *testing.T) {
	t.Run("AllFiltersNil", func(t *testing.T) {
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
		if len(args) != 3 {
			t.Errorf("expected 3 args (original slice), got %d", len(args))
		}
		if args[0] != "alice" {
			t.Errorf("expected args[0] = \"alice\", got %v", args[0])
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("EmailOnly", func(t *testing.T) {
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
		if len(args) != 3 {
			t.Errorf("expected 3 args (original slice), got %d", len(args))
		}
		if args[1] != email {
			t.Errorf("expected args[1] = email pointer, got %v", args[1])
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		var email *string
		phone := strPtr("+1234567890")
		sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})

		if strings.Contains(sql, "AND email") {
			t.Errorf("email condition should be removed when Email is nil, got:\n%s", sql)
		}
		if !strings.Contains(sql, "AND phone") {
			t.Errorf("expected phone condition to be present, got:\n%s", sql)
		}
		if !strings.Contains(sql, "$3") {
			t.Errorf("expected $3 in SQL (original placeholder), got:\n%s", sql)
		}
		if strings.Contains(sql, "$2") {
			t.Errorf("expected no $2 after email removal, got:\n%s", sql)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args (original slice), got %d", len(args))
		}
		if args[2] != phone {
			t.Errorf("expected args[2] = phone pointer, got %v", args[2])
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("AllFilters", func(t *testing.T) {
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
	})

	t.Run("OrderByPreserved", func(t *testing.T) {
		var email, phone *string
		sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone})
		if !strings.Contains(sql, "ORDER BY id ASC") {
			t.Errorf("ORDER BY should always be present, got:\n%s", sql)
		}
	})
}

func TestDynamicSQL_SearchUsers_HasOrders(t *testing.T) {
	t.Run("False", func(t *testing.T) {
		var email, phone *string
		var ordersSince *time.Time
		sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, ordersSince, false})

		if strings.Contains(sql, "AND EXISTS") {
			t.Errorf("AND EXISTS should be removed when HasOrders is false, got:\n%s", sql)
		}
		if strings.Contains(sql, "SELECT 1 FROM orders") {
			t.Errorf("EXISTS subquery body should be removed when HasOrders is false, got:\n%s", sql)
		}
		if strings.Contains(sql, "orders.user_id") {
			t.Errorf("EXISTS subquery body should be fully removed when HasOrders is false, got:\n%s", sql)
		}
		if len(args) != 5 {
			t.Errorf("expected 5 args (original slice), got %d", len(args))
		}
		if args[4] != false {
			t.Errorf("expected args[4] = false, got %v", args[4])
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("True", func(t *testing.T) {
		var email, phone *string
		var ordersSince *time.Time
		sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, ordersSince, true})

		if !strings.Contains(sql, "AND EXISTS") {
			t.Errorf("AND EXISTS should be present when HasOrders is true, got:\n%s", sql)
		}
		if !strings.Contains(sql, "orders.user_id = users.id") {
			t.Errorf("EXISTS subquery body should be present when HasOrders is true, got:\n%s", sql)
		}
		if len(args) != 5 {
			t.Errorf("expected 5 args (original slice), got %d", len(args))
		}
		if args[4] != true {
			t.Errorf("expected args[4] = true, got %v", args[4])
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("True_WithOrdersSince", func(t *testing.T) {
		var email, phone *string
		ts := time.Now()
		sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, &ts, true})

		if !strings.Contains(sql, "AND EXISTS") {
			t.Errorf("expected AND EXISTS, got:\n%s", sql)
		}
		if !strings.Contains(sql, "orders.created_at >= $4") {
			t.Errorf("expected orders_since condition when OrdersSince is set, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("True_NoOrdersSince", func(t *testing.T) {
		var email, phone *string
		var ordersSince *time.Time
		sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, ordersSince, true})

		if !strings.Contains(sql, "AND EXISTS") {
			t.Errorf("expected AND EXISTS, got:\n%s", sql)
		}
		if strings.Contains(sql, "orders.created_at") {
			t.Errorf("orders_since line should be removed when OrdersSince is nil, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("False_WithOrdersSince", func(t *testing.T) {
		var email, phone *string
		ts := time.Now()
		sql, _ := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, &ts, false})

		if strings.Contains(sql, "AND EXISTS") {
			t.Errorf("AND EXISTS should be removed when HasOrders is false, got:\n%s", sql)
		}
		if strings.Contains(sql, "orders.created_at") {
			t.Errorf("orders_since line should also be removed when HasOrders is false, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("AllFilters", func(t *testing.T) {
		email := strPtr("alice@example.com")
		phone := strPtr("+1234567890")
		ts := time.Now()
		sql, args := db.DynamicSQL(db.SearchUsers, []any{"alice", email, phone, &ts, true})

		if !strings.Contains(sql, "AND email") {
			t.Errorf("expected email condition, got:\n%s", sql)
		}
		if !strings.Contains(sql, "AND phone") {
			t.Errorf("expected phone condition, got:\n%s", sql)
		}
		if !strings.Contains(sql, "AND EXISTS") {
			t.Errorf("expected AND EXISTS condition, got:\n%s", sql)
		}
		if !strings.Contains(sql, "orders.created_at >= $4") {
			t.Errorf("expected orders_since condition, got:\n%s", sql)
		}
		if len(args) != 5 {
			t.Errorf("expected 5 args, got %d", len(args))
		}
		t.Logf("SQL:\n%s", sql)
	})
}

// SearchUsersOrdered: dynamic ORDER BY with flag-only params
func TestDynamicSQL_SearchUsersOrdered(t *testing.T) {
	t.Run("NoOrderFlags", func(t *testing.T) {
		var email *string
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", email, false, false})

		if strings.Contains(sql, "created_at DESC") {
			t.Errorf("created_at DESC should be removed when flag is false, got:\n%s", sql)
		}
		if strings.Contains(sql, "name ASC,") {
			t.Errorf("name ASC should be removed when flag is false, got:\n%s", sql)
		}
		if !strings.Contains(sql, "id ASC") {
			t.Errorf("id ASC fallback should always be present, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("CreatedAtDesc", func(t *testing.T) {
		var email *string
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", email, true, false})

		if !strings.Contains(sql, "created_at DESC,") {
			t.Errorf("expected created_at DESC, got:\n%s", sql)
		}
		if strings.Contains(sql, "name ASC,") {
			t.Errorf("name ASC should be removed when flag is false, got:\n%s", sql)
		}
		if !strings.Contains(sql, "id ASC") {
			t.Errorf("id ASC fallback should always be present, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("NameAsc", func(t *testing.T) {
		var email *string
		sql, _ := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", email, false, true})

		if strings.Contains(sql, "created_at DESC") {
			t.Errorf("created_at DESC should be removed when flag is false, got:\n%s", sql)
		}
		if !strings.Contains(sql, "name ASC,") {
			t.Errorf("expected name ASC, got:\n%s", sql)
		}
		if !strings.Contains(sql, "id ASC") {
			t.Errorf("id ASC fallback should always be present, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("AllFlags", func(t *testing.T) {
		email := strPtr("alice@example.com")
		sql, args := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", email, true, true})

		if !strings.Contains(sql, "AND email") {
			t.Errorf("expected email condition, got:\n%s", sql)
		}
		if !strings.Contains(sql, "created_at DESC,") {
			t.Errorf("expected created_at DESC, got:\n%s", sql)
		}
		if !strings.Contains(sql, "name ASC,") {
			t.Errorf("expected name ASC, got:\n%s", sql)
		}
		if !strings.Contains(sql, "id ASC") {
			t.Errorf("id ASC fallback should always be present, got:\n%s", sql)
		}
		if len(args) != 4 {
			t.Errorf("expected 4 args, got %d", len(args))
		}
		t.Logf("SQL:\n%s", sql)
	})
}

// SearchUsersByContact: multi-param :if @email @phone
// Line is included only when ALL listed params are active.
func TestDynamicSQL_SearchUsersByContact(t *testing.T) {
	t.Run("BothNil", func(t *testing.T) {
		var email, phone *string
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", email, phone})

		if strings.Contains(sql, "email = $2 OR phone") {
			t.Errorf("combined condition should be removed when both email and phone are nil, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("EmailOnly", func(t *testing.T) {
		email := strPtr("alice@example.com")
		var phone *string
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", email, phone})

		if strings.Contains(sql, "email = $2 OR phone") {
			t.Errorf("combined condition should be removed when phone is nil, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		var email *string
		phone := strPtr("+1234567890")
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", email, phone})

		if strings.Contains(sql, "email = $2 OR phone") {
			t.Errorf("combined condition should be removed when email is nil, got:\n%s", sql)
		}
		t.Logf("SQL:\n%s", sql)
	})

	t.Run("Both", func(t *testing.T) {
		email := strPtr("alice@example.com")
		phone := strPtr("+1234567890")
		sql, args := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", email, phone})

		if !strings.Contains(sql, "email = $2 OR phone = $3") {
			t.Errorf("expected combined condition when both email and phone are set, got:\n%s", sql)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
		t.Logf("SQL:\n%s", sql)
	})
}
