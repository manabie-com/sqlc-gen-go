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

	// Trailing comma cleanup: when only the first of N annotated items is kept,
	// DynamicSQL must strip the comma that was separating it from the removed item.
	const trailingCommaQuery = "SELECT * FROM t\nORDER BY\n  CASE WHEN $1::bool THEN id END ASC, -- :if $1\n  CASE WHEN $2::bool THEN id END DESC -- :if $2"

	t.Run("TrailingComma/OnlyFirstItem", func(t *testing.T) {
		b := true
		sql, _ := db.DynamicSQL(trailingCommaQuery, []any{&b, nil})
		// The dangling comma after ASC must be stripped
		assertSQL(t, sql, "SELECT * FROM t\nORDER BY\n  CASE WHEN $1::bool THEN id END ASC")
	})

	t.Run("TrailingComma/BothItems", func(t *testing.T) {
		b := true
		sql, _ := db.DynamicSQL(trailingCommaQuery, []any{&b, &b})
		// Both kept → comma between them is valid
		assertSQL(t, sql, "SELECT * FROM t\nORDER BY\n  CASE WHEN $1::bool THEN id END ASC,\n  CASE WHEN $2::bool THEN id END DESC")
	})

	t.Run("TrailingComma/NeitherItem", func(t *testing.T) {
		sql, _ := db.DynamicSQL(trailingCommaQuery, []any{nil, nil})
		// ORDER BY with no items must be removed entirely
		assertSQL(t, sql, "SELECT * FROM t")
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
}
