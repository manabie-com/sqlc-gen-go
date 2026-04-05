package db

import (
	"example/db"
	"testing"
)

func TestGetUserWithLock(t *testing.T) {
	t.Run("LockFalse", func(t *testing.T) {
		// Lock=false → FOR UPDATE line removed, only $1 (ID) passed as SQL arg
		sql, args := db.DynamicSQL(db.GetUserWithLock, []any{int64(1), false})
		assertAbsent(t, sql, "FOR UPDATE")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
		if args[0] != int64(1) {
			t.Errorf("args[0]: got %v, want 1", args[0])
		}
	})

	t.Run("LockTrue", func(t *testing.T) {
		// Lock=true → FOR UPDATE line kept, only $1 (ID) passed as SQL arg (Lock is annotation-only bool)
		sql, args := db.DynamicSQL(db.GetUserWithLock, []any{int64(1), true})
		assertContains(t, sql, "FOR UPDATE")
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})
}
