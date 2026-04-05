package e2e

import (
	"context"
	"testing"

	"example/db"
	setup "example/e2e-setup"
)

func TestGetUserWithLock(t *testing.T) {
	conn := setup.NewDB(t, "../schema.sql")
	ctx := context.Background()
	q := db.NewLockQueries()

	user := setup.InsertUser(t, conn, "lockuser", "lockuser@example.com", nil)

	t.Run("WithoutLock", func(t *testing.T) {
		got, err := q.GetUserWithLock(ctx, conn, db.GetUserWithLockParams{
			ID:   user.ID,
			Lock: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || got.ID != user.ID {
			t.Errorf("got %v, want user %d", got, user.ID)
		}
	})

	t.Run("WithLock", func(t *testing.T) {
		// FOR UPDATE requires a transaction; run inside one so the lock is released cleanly.
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		got, err := q.GetUserWithLock(ctx, tx, db.GetUserWithLockParams{
			ID:   user.ID,
			Lock: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || got.ID != user.ID {
			t.Errorf("got %v, want user %d", got, user.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		got, err := q.GetUserWithLock(ctx, conn, db.GetUserWithLockParams{
			ID:   -1,
			Lock: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
