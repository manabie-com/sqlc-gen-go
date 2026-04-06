package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
)

func TestSourceNameToPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"users.sql", "Users"},
		{"user_orders.sql", "UserOrders"},
		{"product.sql", "Product"},
		{"my-file.sql", "MyFile"},
		{"/some/path/accounts.sql", "Accounts"},
		{"search_results.sql", "SearchResults"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := sourceNameToPrefix(tc.in)
			if got != tc.want {
				t.Errorf("sourceNameToPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestOutputInterfaceMethod(t *testing.T) {
	ctx := &tmplCtx{
		SourceName:         "users.sql",
		EmitPerFileQueries: false,
	}
	if !ctx.OutputInterfaceMethod("anything.sql") {
		t.Error("expected true when EmitPerFileQueries=false")
	}

	ctx.EmitPerFileQueries = true
	if !ctx.OutputInterfaceMethod("users.sql") {
		t.Error("expected true for matching source name")
	}
	if ctx.OutputInterfaceMethod("orders.sql") {
		t.Error("expected false for non-matching source name")
	}
}

func TestFilterUnusedStructs(t *testing.T) {
	usedEnum := Enum{Name: "Status"}
	unusedEnum := Enum{Name: "Role"}
	usedStruct := Struct{Name: "User"}
	unusedStruct := Struct{Name: "Product"}

	queries := []Query{
		{
			Cmd: metadata.CmdOne,
			Arg: QueryValue{
				Typ: "Status",
			},
			Ret: QueryValue{
				Struct: &usedStruct,
			},
		},
	}

	keepEnums, keepStructs := filterUnusedStructs(
		[]Enum{usedEnum, unusedEnum},
		[]Struct{usedStruct, unusedStruct},
		queries,
	)

	if len(keepEnums) != 1 || keepEnums[0].Name != "Status" {
		t.Errorf("keepEnums = %v, want [Status]", keepEnums)
	}
	if len(keepStructs) != 1 || keepStructs[0].Name != "User" {
		t.Errorf("keepStructs = %v, want [User]", keepStructs)
	}
}

func TestFilterUnusedStructs_NullEnum(t *testing.T) {
	statusEnum := Enum{Name: "Status"}
	queries := []Query{
		{
			Cmd: metadata.CmdOne,
			Arg: QueryValue{Typ: "NullStatus"},
			Ret: QueryValue{},
		},
	}
	keepEnums, _ := filterUnusedStructs([]Enum{statusEnum}, []Struct{}, queries)
	if len(keepEnums) != 1 {
		t.Errorf("expected Status enum to be kept via NullStatus, got %v", keepEnums)
	}
}
