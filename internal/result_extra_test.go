package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestParamName_WithName(t *testing.T) {
	p := &plugin.Parameter{
		Column: &plugin.Column{Name: "user_id"},
		Number: 1,
	}
	got := paramName(p)
	if got != "userID" {
		t.Errorf("paramName = %q, want userID", got)
	}
}

func TestParamName_Dollar(t *testing.T) {
	p := &plugin.Parameter{
		Column: &plugin.Column{Name: ""},
		Number: 3,
	}
	got := paramName(p)
	if got != "dollar_3" {
		t.Errorf("paramName = %q, want dollar_3", got)
	}
}

func TestColumnName_WithName(t *testing.T) {
	c := &plugin.Column{Name: "user_id"}
	got := columnName(c, 0)
	if got != "user_id" {
		t.Errorf("columnName = %q, want user_id", got)
	}
}

func TestColumnName_Positional(t *testing.T) {
	c := &plugin.Column{Name: ""}
	got := columnName(c, 2)
	if got != "column_3" {
		t.Errorf("columnName = %q, want column_3", got)
	}
}
