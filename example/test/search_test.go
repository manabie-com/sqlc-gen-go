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

func assertSQL(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("sql mismatch:\ngot:\n%s\n\nwant:\n%s", got, want)
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
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg (only name/$1), got %d", len(a))
		}
	})

	t.Run("EmailOnly", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(strPtr("alice@example.com"), nil, nil, false))
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND email = $2
ORDER BY id ASC`)
		if len(a) != 2 {
			t.Errorf("expected 2 args, got %d", len(a))
		}
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, strPtr("+1234567890"), nil, false))
		// email ($2) removed → phone renumbered from $3 to $2
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND phone = $2
ORDER BY id ASC`)
		if len(a) != 2 {
			t.Errorf("expected 2 args, got %d", len(a))
		}
	})

	t.Run("EmailAndPhone", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(strPtr("a@b.com"), strPtr("+1"), nil, false))
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND email = $2
  AND phone = $3
ORDER BY id ASC`)
		if len(a) != 3 {
			t.Errorf("expected 3 args, got %d", len(a))
		}
	})

	t.Run("HasOrders_False", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, nil, nil, false))
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("HasOrders_True_NoOrdersSince", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, nil, nil, true))
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND EXISTS (
    SELECT 1 FROM orders
    WHERE orders.user_id = users.id
  )
ORDER BY id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("HasOrders_True_WithOrdersSince", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, nil, &now, true))
		// email($2) and phone($3) removed → ordersSince renumbered from $4 to $2
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND EXISTS (
    SELECT 1 FROM orders
    WHERE orders.user_id = users.id
      AND orders.created_at >= $2
  )
ORDER BY id ASC`)
		if len(a) != 2 {
			t.Errorf("expected 2 args ($1=name, $2=ordersSince), got %d", len(a))
		}
	})

	t.Run("HasOrders_False_OrdersSince_Ignored", func(t *testing.T) {
		// OrdersSince is set but HasOrders=false — the whole EXISTS block should be dropped
		sql, a := db.DynamicSQL(db.SearchUsers, args(nil, nil, &now, false))
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("AllFilters", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsers, args(
			strPtr("alice@example.com"), strPtr("+1234567890"), &now, true,
		))
		// All SQL params kept: $1=name, $2=email, $3=phone, $4=ordersSince
		// $5=hasOrders is annotation-only (bool flag), not a SQL placeholder
		assertSQL(t, sql, `-- name: SearchUsers :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND email = $2
  AND phone = $3
  AND EXISTS (
    SELECT 1 FROM orders
    WHERE orders.user_id = users.id
      AND orders.created_at >= $4
  )
ORDER BY id ASC`)
		if len(a) != 4 {
			t.Errorf("expected 4 args ($1-$4), got %d", len(a))
		}
	})
}

func TestSearchUsersOrdered(t *testing.T) {
	t.Run("NoOrderFlags", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, false, false})
		assertSQL(t, sql, `-- name: SearchUsersOrdered :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("CreatedAtDesc", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, true, false})
		assertSQL(t, sql, `-- name: SearchUsersOrdered :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  created_at DESC,
  id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("NameAsc", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", nil, false, true})
		assertSQL(t, sql, `-- name: SearchUsersOrdered :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  name ASC,
  id ASC`)
		if len(a) != 1 {
			t.Errorf("expected 1 arg, got %d", len(a))
		}
	})

	t.Run("AllFlags", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrdered, []any{"alice", strPtr("alice@example.com"), true, true})
		// $3 and $4 are bool-flag annotations only, not SQL placeholders → only $1, $2 returned
		assertSQL(t, sql, `-- name: SearchUsersOrdered :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND email = $2
ORDER BY
  created_at DESC,
  name ASC,
  id ASC`)
		if len(a) != 2 {
			t.Errorf("expected 2 args ($1=name, $2=email), got %d", len(a))
		}
	})
}

func TestSearchUsersByContact(t *testing.T) {
	// Line is kept only when BOTH $2 AND $3 are non-nil.
	noContactSQL := `-- name: SearchUsersByContact :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY id ASC`

	t.Run("BothNil", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", nil, nil})
		assertSQL(t, sql, noContactSQL)
	})

	t.Run("EmailOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", strPtr("alice@example.com"), nil})
		assertSQL(t, sql, noContactSQL)
	})

	t.Run("PhoneOnly", func(t *testing.T) {
		sql, _ := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", nil, strPtr("+1234567890")})
		assertSQL(t, sql, noContactSQL)
	})

	t.Run("Both", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersByContact, []any{"alice", strPtr("alice@example.com"), strPtr("+1234567890")})
		assertSQL(t, sql, `-- name: SearchUsersByContact :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND (email = $2 OR phone = $3)
ORDER BY id ASC`)
		if len(a) != 3 {
			t.Errorf("expected 3 args, got %d", len(a))
		}
	})
}

func TestSearchUsersOrderedByID(t *testing.T) {
	t.Run("IDAsc", func(t *testing.T) {
		// IdAsc/IdDesc are annotation-only bool flags — not SQL placeholders
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, true, false})
		// trailing comma after id ASC must be stripped (id DESC removed)
		assertSQL(t, sql, `-- name: SearchUsersOrderedByID :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  id ASC`)
		if len(a) != 1 {
			t.Errorf("args len: got %d, want 1", len(a))
		}
	})

	t.Run("IDDesc", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, false, true})
		assertSQL(t, sql, `-- name: SearchUsersOrderedByID :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  id DESC`)
		if len(a) != 1 {
			t.Errorf("args len: got %d, want 1", len(a))
		}
	})

	t.Run("BothFlags", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", nil, true, true})
		assertSQL(t, sql, `-- name: SearchUsersOrderedByID :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
ORDER BY
  id ASC,
  id DESC`)
		if len(a) != 1 {
			t.Errorf("args len: got %d, want 1", len(a))
		}
	})

	t.Run("WithEmail", func(t *testing.T) {
		sql, a := db.DynamicSQL(db.SearchUsersOrderedByID, []any{"alice", strPtr("alice@example.com"), true, true})
		// $1=name, $2=email; bool flags are annotation-only
		assertSQL(t, sql, `-- name: SearchUsersOrderedByID :many
SELECT id, name, email, created_at, phone FROM users
WHERE name = $1
  AND email = $2
ORDER BY
  id ASC,
  id DESC`)
		if len(a) != 2 {
			t.Errorf("args len: got %d, want 2", len(a))
		}
	})
}
