package db

import (
	"example/db"
	"testing"
)

func TestDynamicSQL(t *testing.T) {
	t.Run("RemapsPlaceholders", func(t *testing.T) {
		// $1 required, $2 conditional (nil → line removed)
		query := "SELECT * FROM t\nWHERE a = $1\n  AND b = $2 -- :if $2"
		var b *string
		gotQuery, gotArgs := db.DynamicSQL(query, []any{"hello", b})

		assertSQL(t, gotQuery, "SELECT * FROM t\nWHERE a = $1")
		if len(gotArgs) != 1 {
			t.Errorf("args len: got %d, want 1", len(gotArgs))
		}
		if gotArgs[0] != "hello" {
			t.Errorf("args[0]: got %v, want hello", gotArgs[0])
		}
	})

	t.Run("RemapsGaps", func(t *testing.T) {
		// $1 required, $2 conditional removed, $3 required → $3 renumbered to $2
		query := "SELECT * FROM t\nWHERE a = $1\n  AND b = $2 -- :if $2\n  AND c = $3"
		var b *string
		gotQuery, gotArgs := db.DynamicSQL(query, []any{"a", b, "c"})

		assertSQL(t, gotQuery, "SELECT * FROM t\nWHERE a = $1\n  AND c = $2")
		if len(gotArgs) != 2 {
			t.Fatalf("args len: got %d, want 2", len(gotArgs))
		}
		if gotArgs[0] != "a" || gotArgs[1] != "c" {
			t.Errorf("args: got %v, want [a c]", gotArgs)
		}
	})

	t.Run("AllConditionsActive", func(t *testing.T) {
		// All lines kept → placeholders stay sequential, args unchanged
		status := "active"
		query := "SELECT * FROM t\nWHERE a = $1\n  AND b = $2 -- :if $2"
		gotQuery, gotArgs := db.DynamicSQL(query, []any{"hello", &status})

		assertSQL(t, gotQuery, "SELECT * FROM t\nWHERE a = $1\n  AND b = $2")
		if len(gotArgs) != 2 {
			t.Fatalf("args len: got %d, want 2", len(gotArgs))
		}
	})

	t.Run("NoAnnotations", func(t *testing.T) {
		query := "SELECT * FROM t WHERE a = $1 AND b = $2"
		gotQuery, gotArgs := db.DynamicSQL(query, []any{"x", "y"})

		assertSQL(t, gotQuery, "SELECT * FROM t WHERE a = $1 AND b = $2")
		if len(gotArgs) != 2 {
			t.Errorf("args len: got %d, want 2", len(gotArgs))
		}
	})

	// ORDER BY flag tests: $1=required WHERE param, $2/$3=bool flags for ORDER BY lines
	const orderByQuery = "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC, -- :if $2\n  name ASC, -- :if $3\n  created_at DESC"

	t.Run("OrderBy/AllFlagsInactive", func(t *testing.T) {
		sql, args := db.DynamicSQL(orderByQuery, []any{"x", false, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  created_at DESC")
		// $2 and $3 are annotation-only flags → only $1 remains
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/FirstFlagActive", func(t *testing.T) {
		sql, args := db.DynamicSQL(orderByQuery, []any{"x", true, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC,\n  created_at DESC")
		// $2 is annotation-only → only $1 in args
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/AllFlagsActive", func(t *testing.T) {
		sql, args := db.DynamicSQL(orderByQuery, []any{"x", true, true})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC,\n  name ASC,\n  created_at DESC")
		// $2 and $3 are annotation-only flags → only $1 in args
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/RemovedWhenEmpty", func(t *testing.T) {
		query := "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC, -- :if $2\n  id DESC -- :if $3"
		sql, args := db.DynamicSQL(query, []any{"x", false, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/KeptWhenHasContent", func(t *testing.T) {
		query := "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC, -- :if $2\n  id DESC -- :if $3"
		sql, _ := db.DynamicSQL(query, []any{"x", false, true})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id DESC")
	})

	t.Run("OrderBy/KeptWhenHasContentFirstItem", func(t *testing.T) {
		query := "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC, -- :if $2\n  id DESC -- :if $3"
		sql, _ := db.DynamicSQL(query, []any{"x", true, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC")
	})

	const orderByOnlyConditionalQuery = "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC, -- :if $2\n  name DESC, -- :if $3\n  created_at DESC -- :if $4"

	t.Run("OrderBy/OmitAllLines", func(t *testing.T) {
		sql, args := db.DynamicSQL(orderByOnlyConditionalQuery, []any{"x", false, false, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/ContainsSecondLine", func(t *testing.T) {
		sql, args := db.DynamicSQL(orderByOnlyConditionalQuery, []any{"x", false, true, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  name DESC")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("OrderBy/RemoveAllLinesSimple", func(t *testing.T) {
		query := "SELECT * FROM t\nWHERE a = $1\nORDER BY\n  id ASC -- :if $2\n  id DESC -- :if $3"
		sql, args := db.DynamicSQL(query, []any{"x", false, false})
		assertSQL(t, sql, "SELECT * FROM t\nWHERE a = $1")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})
}
