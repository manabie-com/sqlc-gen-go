package db

import (
	"example/db"
	"testing"
)

func TestGetUserWithLock(t *testing.T) {
	t.Run("LockFalse", func(t *testing.T) {
		// Lock=false → FOR UPDATE line removed, only $1 (ID) passed as SQL arg
		sql, args := db.DynamicSQL(db.GetUserWithLock, []any{int64(1), false})
		assertSQL(t, sql, `-- name: GetUserWithLock :one
SELECT id, name, email, created_at, phone
FROM users
WHERE id = $1
LIMIT 1`)
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})

	t.Run("LockTrue", func(t *testing.T) {
		// Lock=true → FOR UPDATE line kept, only $1 (ID) passed as SQL arg (Lock is annotation-only bool)
		sql, args := db.DynamicSQL(db.GetUserWithLock, []any{int64(1), true})
		assertSQL(t, sql, `-- name: GetUserWithLock :one
SELECT id, name, email, created_at, phone
FROM users
WHERE id = $1
LIMIT 1
FOR UPDATE`)
		if len(args) != 1 {
			t.Errorf("args len: got %d, want 1", len(args))
		}
	})
}
