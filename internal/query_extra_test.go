package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestQueryValue_SlicePair(t *testing.T) {
	v := QueryValue{Name: "arg", Typ: "int32"}
	got := v.SlicePair()
	if got != "arg []int32" {
		t.Errorf("SlicePair() = %q, want %q", got, "arg []int32")
	}
}

func TestQueryValue_SlicePair_Empty(t *testing.T) {
	v := QueryValue{}
	got := v.SlicePair()
	if got != "" {
		t.Errorf("SlicePair() on empty = %q, want empty", got)
	}
}

func TestQueryValue_UniqueFields(t *testing.T) {
	s := &Struct{
		Fields: []Field{
			{Name: "ID", Type: "int32"},
			{Name: "Name", Type: "string"},
			{Name: "ID", Type: "int32"}, // duplicate
		},
	}
	v := QueryValue{Struct: s}
	fields := v.UniqueFields()
	if len(fields) != 2 {
		t.Errorf("UniqueFields() returned %d fields, want 2", len(fields))
	}
	if fields[0].Name != "ID" || fields[1].Name != "Name" {
		t.Errorf("UniqueFields() = %v", fields)
	}
}

func TestQueryValue_ColumnNames_Scalar(t *testing.T) {
	v := QueryValue{DBName: "user_id"}
	names := v.ColumnNames()
	if len(names) != 1 || names[0] != "user_id" {
		t.Errorf("ColumnNames() = %v, want [user_id]", names)
	}
}

func TestQueryValue_ColumnNames_Struct(t *testing.T) {
	v := QueryValue{
		Struct: &Struct{
			Fields: []Field{
				{DBName: "id"},
				{DBName: "name"},
			},
		},
	}
	names := v.ColumnNames()
	if len(names) != 2 || names[0] != "id" || names[1] != "name" {
		t.Errorf("ColumnNames() = %v, want [id name]", names)
	}
}

func TestQueryValue_CopyFromMySQLFields_Struct(t *testing.T) {
	s := &Struct{
		Fields: []Field{
			{Name: "ID", Type: "int32"},
		},
	}
	v := QueryValue{Struct: s}
	fields := v.CopyFromMySQLFields()
	if len(fields) != 1 || fields[0].Name != "ID" {
		t.Errorf("CopyFromMySQLFields() = %v", fields)
	}
}

func TestQueryValue_CopyFromMySQLFields_Scalar(t *testing.T) {
	v := QueryValue{Name: "id", DBName: "id", Typ: "int32"}
	fields := v.CopyFromMySQLFields()
	if len(fields) != 1 || fields[0].Name != "id" {
		t.Errorf("CopyFromMySQLFields() = %v", fields)
	}
}

func TestQueryValue_VariableForField_NonStruct(t *testing.T) {
	v := QueryValue{Name: "arg", Typ: "int32"}
	f := Field{Name: "ID"}
	got := v.VariableForField(f)
	if got != "arg" {
		t.Errorf("VariableForField (non-struct) = %q, want arg", got)
	}
}

func TestQueryValue_VariableForField_Struct_NoEmit(t *testing.T) {
	v := QueryValue{
		Name:   "arg",
		Struct: &Struct{Name: "Arg"},
		// Emit is false (default)
	}
	f := Field{Name: "UserID"}
	got := v.VariableForField(f)
	if got != "userID" {
		t.Errorf("VariableForField (struct, no emit) = %q, want userID", got)
	}
}

func TestQueryValue_VariableForField_Struct_Emit(t *testing.T) {
	v := QueryValue{
		Name:   "arg",
		Struct: &Struct{Name: "Arg"},
		Emit:   true,
	}
	f := Field{Name: "UserID"}
	got := v.VariableForField(f)
	if got != "arg.UserID" {
		t.Errorf("VariableForField (struct, emit) = %q, want arg.UserID", got)
	}
}

func TestQueryValue_Params_Scalar(t *testing.T) {
	v := QueryValue{
		Name:      "id",
		Typ:       "int32",
		Column:    &plugin.Column{},
		SQLDriver: "github.com/jackc/pgx/v5",
	}
	got := v.Params()
	if got != "id" {
		t.Errorf("Params() = %q, want id", got)
	}
}
