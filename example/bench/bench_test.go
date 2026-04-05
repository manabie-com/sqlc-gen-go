package bench

import (
	"example/db"
	"fmt"
	"strings"
	"testing"
	"time"
)

func strPtr(v string) *string { return &v }

func manualSearchUsers(name string, email, phone *string, ordersSince *time.Time, hasOrders bool) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, 5)
	idx := 1

	sb.WriteString("SELECT id, name, email, created_at, phone FROM users\nWHERE name = $1")
	args = append(args, name)
	idx++

	if email != nil {
		fmt.Fprintf(&sb, "\n  AND email = $%d", idx)
		args = append(args, email)
		idx++
	}
	if phone != nil {
		fmt.Fprintf(&sb, "\n  AND phone = $%d", idx)
		args = append(args, phone)
		idx++
	}
	if hasOrders {
		if ordersSince != nil {
			fmt.Fprintf(&sb, "\n  AND EXISTS (\n    SELECT 1 FROM orders\n    WHERE orders.user_id = users.id\n      AND orders.created_at >= $%d\n  )", idx)
			args = append(args, ordersSince)
		} else {
			sb.WriteString("\n  AND EXISTS (\n    SELECT 1 FROM orders\n    WHERE orders.user_id = users.id\n  )")
		}
	}
	sb.WriteString("\nORDER BY id ASC")
	return sb.String(), args
}

var (
	benchEmail = strPtr("alice@example.com")
	benchPhone = strPtr("+1234567890")
	benchTime  = func() *time.Time { t := time.Now(); return &t }()
)

func BenchmarkDynamicSQL_NoOptional(b *testing.B) {
	for b.Loop() {
		db.DynamicSQL(db.SearchUsers, []any{"alice", (*string)(nil), (*string)(nil), (*time.Time)(nil), false})
	}
}

func BenchmarkManual_NoOptional(b *testing.B) {
	for b.Loop() {
		manualSearchUsers("alice", nil, nil, nil, false)
	}
}

func BenchmarkDynamicSQL_AllOptional(b *testing.B) {
	for b.Loop() {
		db.DynamicSQL(db.SearchUsers, []any{"alice", benchEmail, benchPhone, benchTime, true})
	}
}

func BenchmarkManual_AllOptional(b *testing.B) {
	for b.Loop() {
		manualSearchUsers("alice", benchEmail, benchPhone, benchTime, true)
	}
}
