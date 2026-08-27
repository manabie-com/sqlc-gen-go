package e2emysql

import (
	"context"
	"reflect"
	"testing"

	"example/dbmysql"
)

func TestDynamicFilter(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	q := dbmysql.New(db)

	insert := func(kind, a, b, c string) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx,
			"INSERT INTO filter_items (kind, a, b, c) VALUES (?, ?, ?, ?)", kind, a, b, c)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	t.Cleanup(func() { db.Exec("DELETE FROM filter_items") })

	insert("widget", "required-a", "first", "required-c")
	id2 := insert("widget", "required-a", "second", "required-c")
	insert("gadget", "required-a", "first", "other-c")

	t.Run("ListFilterItems/OptionalMiddleArgOmitted", func(t *testing.T) {
		items, err := q.ListFilterItems(ctx, dbmysql.ListFilterItemsParams{
			A: "required-a",
			C: "required-c",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Errorf("nil b: got %d rows, want 2", len(items))
		}
	})

	t.Run("ListFilterItems/OptionalMiddleArgSupplied", func(t *testing.T) {
		// b is dropped or kept between two required params, so @c is renumbered.
		items, err := q.ListFilterItems(ctx, dbmysql.ListFilterItemsParams{
			A: "required-a",
			B: strPtr("first"),
			C: "required-c",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].B != "first" {
			t.Errorf("non-nil b: got %v, want only the b=first row", items)
		}
	})

	t.Run("SearchFilterItems/NilIDs", func(t *testing.T) {
		ids, err := q.SearchFilterItems(ctx, dbmysql.SearchFilterItemsParams{Kind: "widget"})
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 2 {
			t.Errorf("nil ids: got %v, want 2 rows", ids)
		}
	})

	t.Run("SearchFilterItems/SpecificIDs", func(t *testing.T) {
		ids, err := q.SearchFilterItems(ctx, dbmysql.SearchFilterItemsParams{
			Kind: "widget",
			Ids:  []int64{id2, id2 + 1000},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ids, []int64{id2}) {
			t.Errorf("ids filter: got %v, want [%d]", ids, id2)
		}
	})

	t.Run("SearchFilterItems/EmptySliceMatchesNothing", func(t *testing.T) {
		// empty non-nil slice → condition active → IN (NULL) → zero rows
		ids, err := q.SearchFilterItems(ctx, dbmysql.SearchFilterItemsParams{
			Kind: "widget",
			Ids:  []int64{},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 0 {
			t.Errorf("empty ids: got %v, want 0 rows (filter by empty set)", ids)
		}
	})

	t.Run("SearchFilterItems/NilableSliceSkipsCondition", func(t *testing.T) {
		// NilableSlice(empty) → nil → clause skipped for callers who want
		// empty to mean "don't filter"
		ids, err := q.SearchFilterItems(ctx, dbmysql.SearchFilterItemsParams{
			Kind: "widget",
			Ids:  dbmysql.NilableSlice([]int64{}),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 2 {
			t.Errorf("NilableSlice(empty) ids: got %v, want 2 rows (clause skipped)", ids)
		}
	})
}
