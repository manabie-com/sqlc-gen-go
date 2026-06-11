package golang

import (
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// ---- argName ----------------------------------------------------------------

func TestArgName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"name", "name"},
		{"user_id", "userID"},
		{"first_name", "firstName"},
		{"order_created_at", "orderCreatedAt"},
		{"api_key", "apiKey"},
		{"id", "id"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := argName(tc.in)
			if got != tc.want {
				t.Errorf("argName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ---- Query.IsSelect ---------------------------------------------------------

func TestQuery_IsSelect(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{
		{"SELECT * FROM t", true},
		{"select * from t", true},
		{"  SELECT id FROM users", true},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"with cte as (select 1) select 1", true},
		{"INSERT INTO t (a) VALUES ($1)", false},
		{"UPDATE t SET a = $1", false},
		{"DELETE FROM t WHERE id = $1", false},
	}
	for _, tc := range cases {
		label := tc.sql
		if len(label) > 20 {
			label = label[:20]
		}
		t.Run(label, func(t *testing.T) {
			q := Query{SQL: tc.sql}
			got := q.IsSelect()
			if got != tc.want {
				t.Errorf("IsSelect(%q) = %v, want %v", tc.sql, got, tc.want)
			}
		})
	}
}

// ---- Query.TableIdentifierAsGoSlice / TableIdentifierForMySQL ---------------

func TestQuery_TableIdentifierAsGoSlice(t *testing.T) {
	cases := []struct {
		catalog, schema, name string
		want                  string
	}{
		{"", "", "users", `[]string{"users"}`},
		{"", "public", "users", `[]string{"public", "users"}`},
		{"mydb", "public", "users", `[]string{"mydb", "public", "users"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := Query{Table: &plugin.Identifier{Catalog: tc.catalog, Schema: tc.schema, Name: tc.name}}
			got := q.TableIdentifierAsGoSlice()
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestQuery_TableIdentifierForMySQL(t *testing.T) {
	cases := []struct {
		catalog, schema, name string
		want                  string
	}{
		{"", "", "users", "`users`"},
		{"", "mydb", "orders", "`mydb`.`orders`"},
		{"cat", "sch", "tbl", "`cat`.`sch`.`tbl`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := Query{Table: &plugin.Identifier{Catalog: tc.catalog, Schema: tc.schema, Name: tc.name}}
			got := q.TableIdentifierForMySQL()
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ---- QueryValue predicates --------------------------------------------------

func TestQueryValue_isEmpty(t *testing.T) {
	if !(QueryValue{}).isEmpty() {
		t.Error("zero-value QueryValue should be empty")
	}
	if (QueryValue{Typ: "string"}).isEmpty() {
		t.Error("QueryValue with Typ should not be empty")
	}
	if (QueryValue{Name: "x"}).isEmpty() {
		t.Error("QueryValue with Name should not be empty")
	}
	if (QueryValue{Struct: &Struct{}}).isEmpty() {
		t.Error("QueryValue with Struct should not be empty")
	}
}

func TestQueryValue_IsPointer(t *testing.T) {
	// IsPointer = EmitPointer && Struct != nil
	s := &Struct{Name: "MyStruct"}
	if (QueryValue{EmitPointer: true, Struct: s}).IsPointer() != true {
		t.Error("expected IsPointer() = true when EmitPointer=true and Struct set")
	}
	if (QueryValue{EmitPointer: false, Struct: s}).IsPointer() != false {
		t.Error("expected IsPointer() = false when EmitPointer=false")
	}
	if (QueryValue{EmitPointer: true}).IsPointer() != false {
		t.Error("expected IsPointer() = false when Struct is nil")
	}
}

func TestQueryValue_Type(t *testing.T) {
	// Typ takes priority
	if got := (QueryValue{Typ: "int64", Struct: &Struct{Name: "S"}}).Type(); got != "int64" {
		t.Errorf("expected Typ to take priority, got %q", got)
	}
	// Fallback to Struct.Name
	if got := (QueryValue{Struct: &Struct{Name: "MyRow"}}).Type(); got != "MyRow" {
		t.Errorf("expected Struct.Name, got %q", got)
	}
}

func TestQueryValue_DefineType(t *testing.T) {
	s := &Struct{Name: "Item"}
	// pointer
	if got := (&QueryValue{EmitPointer: true, Struct: s}).DefineType(); got != "*Item" {
		t.Errorf("expected *Item, got %q", got)
	}
	// non-pointer struct
	if got := (&QueryValue{Struct: s}).DefineType(); got != "Item" {
		t.Errorf("expected Item, got %q", got)
	}
	// plain type
	if got := (&QueryValue{Typ: "string"}).DefineType(); got != "string" {
		t.Errorf("expected string, got %q", got)
	}
}

func TestQueryValue_SlicePointer(t *testing.T) {
	s := &Struct{Name: "Item"}

	// EmitPointer without DisableSlicePointer: slice elements are pointers,
	// matching DefineType/ReturnName.
	ptr := QueryValue{EmitPointer: true, Struct: s, Name: "i"}
	if !ptr.IsSlicePointer() {
		t.Error("expected IsSlicePointer() = true when EmitPointer set and slice pointers not disabled")
	}
	if got := ptr.SliceType(); got != "*Item" {
		t.Errorf("expected *Item, got %q", got)
	}
	if got := ptr.SliceReturnName(); got != "&i" {
		t.Errorf("expected &i, got %q", got)
	}

	// DisableSlicePointer keeps slice elements as values, while the single-row
	// DefineType/ReturnName stay pointers.
	noSlice := QueryValue{EmitPointer: true, DisableSlicePointer: true, Struct: s, Name: "i"}
	if noSlice.IsSlicePointer() {
		t.Error("expected IsSlicePointer() = false when DisableSlicePointer set")
	}
	if got := noSlice.SliceType(); got != "Item" {
		t.Errorf("expected Item, got %q", got)
	}
	if got := noSlice.SliceReturnName(); got != "i" {
		t.Errorf("expected i, got %q", got)
	}
	if got := noSlice.DefineType(); got != "*Item" {
		t.Errorf("expected DefineType to stay *Item, got %q", got)
	}
	if got := noSlice.ReturnName(); got != "&i" {
		t.Errorf("expected ReturnName to stay &i, got %q", got)
	}

	// Without EmitPointer, DisableSlicePointer is a no-op.
	plain := QueryValue{DisableSlicePointer: true, Struct: s, Name: "i"}
	if plain.IsSlicePointer() {
		t.Error("expected IsSlicePointer() = false when EmitPointer unset")
	}
	if got := plain.SliceType(); got != "Item" {
		t.Errorf("expected Item, got %q", got)
	}
}

func TestQueryValue_ColumnNamesAsGoSlice(t *testing.T) {
	// nil struct: uses DBName
	v := QueryValue{DBName: "user_id"}
	got := v.ColumnNamesAsGoSlice()
	if got != `[]string{"user_id"}` {
		t.Errorf("got %q", got)
	}

	// struct fields: uses OriginalName when set, else DBName
	v2 := QueryValue{Struct: &Struct{
		Fields: []Field{
			{DBName: "id", Column: &plugin.Column{OriginalName: "orig_id"}},
			{DBName: "name"},
		},
	}}
	got2 := v2.ColumnNamesAsGoSlice()
	if !strings.Contains(got2, `"orig_id"`) {
		t.Errorf("expected orig_id, got %q", got2)
	}
	if !strings.Contains(got2, `"name"`) {
		t.Errorf("expected name, got %q", got2)
	}
}

// ---- checkIncompatibleFieldTypes --------------------------------------------

func TestCheckIncompatibleFieldTypes(t *testing.T) {
	t.Run("Compatible_SameType", func(t *testing.T) {
		fields := []Field{
			{Name: "Status", Type: "string"},
			{Name: "Status", Type: "string"},
		}
		if err := checkIncompatibleFieldTypes(fields); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Incompatible_DifferentTypes", func(t *testing.T) {
		fields := []Field{
			{Name: "Status", Type: "string"},
			{Name: "Status", Type: "int32"},
		}
		err := checkIncompatibleFieldTypes(fields)
		if err == nil {
			t.Error("expected error for incompatible types")
		}
		if !strings.Contains(err.Error(), "Status") {
			t.Errorf("error should mention field name, got: %v", err)
		}
	})

	t.Run("NoDuplicates", func(t *testing.T) {
		fields := []Field{
			{Name: "ID", Type: "int64"},
			{Name: "Name", Type: "string"},
		}
		if err := checkIncompatibleFieldTypes(fields); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		if err := checkIncompatibleFieldTypes(nil); err != nil {
			t.Errorf("unexpected error for empty fields: %v", err)
		}
	})
}
