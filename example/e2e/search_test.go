package e2e

import (
	"context"
	"testing"
	"time"

	"example/db"
	setup "example/e2e-setup"
)

func TestSearchUsers(t *testing.T) {
	conn := setup.NewDB(t, "../schema.sql")
	ctx := context.Background()
	q := db.NewSearchQueries()

	alice1 := setup.InsertUser(t, conn, "alice", "alice@example.com", setup.StrPtr("+1111111111"))
	alice2 := setup.InsertUser(t, conn, "alice", "alice2@example.com", nil)
	bob := setup.InsertUser(t, conn, "bob", "bob@example.com", nil)
	setup.InsertOrder(t, conn, alice1.ID, time.Now().Add(-24*time.Hour))

	t.Run("NoOptionalFilters", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{Name: "alice"})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	t.Run("EmailFilter", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:  "alice",
			Email: setup.StrPtr("alice@example.com"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != alice1.ID {
			t.Errorf("got %v, want alice1", users)
		}
	})

	t.Run("PhoneFilter", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:  "alice",
			Phone: setup.StrPtr("+1111111111"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != alice1.ID {
			t.Errorf("got %v, want alice1", users)
		}
	})

	t.Run("HasOrders_False", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:      "alice",
			HasOrders: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	t.Run("HasOrders_True", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:      "alice",
			HasOrders: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != alice1.ID {
			t.Errorf("got %v, want only alice1 (has order)", users)
		}
	})

	t.Run("HasOrders_True_WithOrdersSince_Match", func(t *testing.T) {
		since := time.Now().Add(-48 * time.Hour)
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:        "alice",
			HasOrders:   true,
			OrdersSince: &since,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != alice1.ID {
			t.Errorf("got %v, want alice1", users)
		}
	})

	t.Run("HasOrders_True_WithOrdersSince_NoMatch", func(t *testing.T) {
		since := time.Now().Add(time.Hour) // future — no orders qualify
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{
			Name:        "alice",
			HasOrders:   true,
			OrdersSince: &since,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 0 {
			t.Errorf("got %d users, want 0", len(users))
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		users, err := q.SearchUsers(ctx, conn, db.SearchUsersParams{Name: bob.Name})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != bob.ID {
			t.Errorf("got %v, want bob", users)
		}
	})

	_ = alice2
}

func TestSearchUsersOrdered(t *testing.T) {
	conn := setup.NewDB(t, "../schema.sql")
	ctx := context.Background()
	q := db.NewSearchQueries()

	// Insert two alices so ordering is observable
	a1 := setup.InsertUser(t, conn, "alice", "alice.a@example.com", nil)
	a2 := setup.InsertUser(t, conn, "alice", "alice.b@example.com", nil)

	t.Run("NoOrderFlags", func(t *testing.T) {
		users, err := q.SearchUsersOrdered(ctx, conn, db.SearchUsersOrderedParams{Name: "alice"})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
		// default order is id ASC
		if users[0].ID > users[1].ID {
			t.Errorf("expected id ASC, got %d > %d", users[0].ID, users[1].ID)
		}
	})

	t.Run("EmailFilter", func(t *testing.T) {
		users, err := q.SearchUsersOrdered(ctx, conn, db.SearchUsersOrderedParams{
			Name:  "alice",
			Email: setup.StrPtr("alice.a@example.com"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != a1.ID {
			t.Errorf("got %v, want a1", users)
		}
	})

	t.Run("OrderCreatedAtDesc", func(t *testing.T) {
		users, err := q.SearchUsersOrdered(ctx, conn, db.SearchUsersOrderedParams{
			Name:               "alice",
			OrderCreatedAtDesc: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	t.Run("OrderNameAsc", func(t *testing.T) {
		users, err := q.SearchUsersOrdered(ctx, conn, db.SearchUsersOrderedParams{
			Name:         "alice",
			OrderNameAsc: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	_ = a2
}

func TestSearchUsersByContact(t *testing.T) {
	conn := setup.NewDB(t, "../schema.sql")
	ctx := context.Background()
	q := db.NewSearchQueries()

	alice := setup.InsertUser(t, conn, "alice", "alice@example.com", setup.StrPtr("+1111111111"))
	_ = setup.InsertUser(t, conn, "alice", "other@example.com", setup.StrPtr("+9999999999"))

	t.Run("BothNil", func(t *testing.T) {
		users, err := q.SearchUsersByContact(ctx, conn, db.SearchUsersByContactParams{Name: "alice"})
		if err != nil {
			t.Fatal(err)
		}
		// no contact filter → all alices returned
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	t.Run("EmailAndPhone_Match", func(t *testing.T) {
		users, err := q.SearchUsersByContact(ctx, conn, db.SearchUsersByContactParams{
			Name:  "alice",
			Email: setup.StrPtr("alice@example.com"),
			Phone: setup.StrPtr("+1111111111"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != alice.ID {
			t.Errorf("got %v, want alice", users)
		}
	})

	t.Run("EmailAndPhone_NoMatch", func(t *testing.T) {
		users, err := q.SearchUsersByContact(ctx, conn, db.SearchUsersByContactParams{
			Name:  "alice",
			Email: setup.StrPtr("nobody@example.com"),
			Phone: setup.StrPtr("+0000000000"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 0 {
			t.Errorf("got %d users, want 0", len(users))
		}
	})
}

func TestSearchUsersOrderedByID(t *testing.T) {
	conn := setup.NewDB(t, "../schema.sql")
	ctx := context.Background()
	q := db.NewSearchQueries()

	a1 := setup.InsertUser(t, conn, "alice", "alice.x@example.com", nil)
	a2 := setup.InsertUser(t, conn, "alice", "alice.y@example.com", nil)

	// Note: NoOrderFlags is not tested here — removing all flags leaves an empty
	// "ORDER BY" clause (syntax error) because the query has no fallback line.
	// That case is covered by unit tests in example/test.

	t.Run("IDAsc", func(t *testing.T) {
		users, err := q.SearchUsersOrderedByID(ctx, conn, db.SearchUsersOrderedByIDParams{
			Name:  "alice",
			IDAsc: setup.BoolPtr(true),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Fatalf("got %d users, want 2", len(users))
		}
		if users[0].ID != a1.ID || users[1].ID != a2.ID {
			t.Errorf("expected id ASC order: got [%d, %d]", users[0].ID, users[1].ID)
		}
	})

	t.Run("IDDesc", func(t *testing.T) {
		users, err := q.SearchUsersOrderedByID(ctx, conn, db.SearchUsersOrderedByIDParams{
			Name:   "alice",
			IDDesc: setup.BoolPtr(true),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Fatalf("got %d users, want 2", len(users))
		}
		if users[0].ID != a2.ID || users[1].ID != a1.ID {
			t.Errorf("expected id DESC order: got [%d, %d]", users[0].ID, users[1].ID)
		}
	})

	t.Run("IDAscAndIDDesc", func(t *testing.T) {
		// Both flags active: ASC line has its trailing comma followed by the DESC line — valid SQL.
		users, err := q.SearchUsersOrderedByID(ctx, conn, db.SearchUsersOrderedByIDParams{
			Name:   "alice",
			IDAsc:  setup.BoolPtr(true),
			IDDesc: setup.BoolPtr(true),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 2 {
			t.Errorf("got %d users, want 2", len(users))
		}
	})

	t.Run("WithEmailFilter", func(t *testing.T) {
		users, err := q.SearchUsersOrderedByID(ctx, conn, db.SearchUsersOrderedByIDParams{
			Name:   "alice",
			Email:  setup.StrPtr("alice.x@example.com"),
			IDAsc:  setup.BoolPtr(true),
			IDDesc: setup.BoolPtr(true),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != a1.ID {
			t.Errorf("got %v, want a1", users)
		}
	})
}
