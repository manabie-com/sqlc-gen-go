package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func testCatalog() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"users": {
			"id":         {},
			"name":       {},
			"email":      {},
			"created_at": {},
		},
		"orders": {
			"id":         {},
			"user_id":    {},
			"amount":     {},
			"status":     {},
			"created_at": {},
		},
	}
}

func TestValidateQueryCTEs_ValidReferences(t *testing.T) {
	tables := testCatalog()

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "simple CTE with correct reference",
			sql: `WITH order_totals AS (
				SELECT user_id, COUNT(*) as order_count
				FROM orders
				GROUP BY user_id
			)
			SELECT u.id, ot.order_count
			FROM users u
			INNER JOIN order_totals ot ON ot.user_id = u.id`,
		},
		{
			name: "multiple CTEs with cross-references",
			sql: `WITH order_totals AS (
				SELECT user_id, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_spent
				FROM orders
				GROUP BY user_id
			),
			user_info AS (
				SELECT u.id, u.name, ot.order_count, ot.total_spent
				FROM users u
				INNER JOIN order_totals ot ON ot.user_id = u.id
			)
			SELECT * FROM user_info
			ORDER BY total_spent DESC`,
		},
		{
			name: "CTE with aliased columns",
			sql: `WITH ranked AS (
				SELECT u.id as user_id, u.name, COALESCE(SUM(o.amount), 0) as total_spent
				FROM users u
				LEFT JOIN orders o ON o.user_id = u.id
				GROUP BY u.id, u.name
			)
			SELECT ranked.user_id, ranked.name, ranked.total_spent FROM ranked`,
		},
		{
			name: "writable CTE with RETURNING *",
			sql: `WITH deleted AS (
				DELETE FROM orders WHERE user_id = $1 RETURNING *
			)
			SELECT deleted.id, deleted.user_id, deleted.amount FROM deleted`,
		},
		{
			name: "INSERT CTE with RETURNING *",
			sql: `WITH upserted AS (
				INSERT INTO users (name, email)
				VALUES ($1, $2)
				ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name
				RETURNING *
			)
			SELECT * FROM upserted`,
		},
		{
			name: "no CTE query",
			sql:  `SELECT id, name FROM users WHERE id = $1`,
		},
		{
			name: "CTE with function expressions",
			sql: `WITH stats AS (
				SELECT user_id, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_spent, MAX(created_at) as last_order_at
				FROM orders
				GROUP BY user_id
			)
			SELECT stats.user_id, stats.order_count, stats.total_spent, stats.last_order_at FROM stats`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryCTEs(tt.sql, tt.name, tables)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateQueryCTEs_InvalidReferences(t *testing.T) {
	tables := testCatalog()

	tests := []struct {
		name      string
		sql       string
		wantField string
		wantCTE   string
	}{
		{
			name: "typo in CTE column reference",
			sql: `WITH user_order_stats AS (
				SELECT orders.user_id, COUNT(*) as order_count
				FROM orders
				GROUP BY orders.user_id
			)
			SELECT users.id
			FROM users
			LEFT JOIN user_order_stats ON user_order_stats.user_id1 = users.id`,
			wantField: "user_id1",
			wantCTE:   "user_order_stats",
		},
		{
			name: "nonexistent column in CTE",
			sql: `WITH totals AS (
				SELECT user_id, SUM(amount) as total
				FROM orders
				GROUP BY user_id
			)
			SELECT totals.nonexistent FROM totals`,
			wantField: "nonexistent",
			wantCTE:   "totals",
		},
		{
			name: "wrong column in RETURNING * CTE",
			sql: `WITH deleted AS (
				DELETE FROM orders WHERE user_id = $1 RETURNING *
			)
			SELECT deleted.wrong_col FROM deleted`,
			wantField: "wrong_col",
			wantCTE:   "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryCTEs(tt.sql, tt.name, tables)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			errStr := err.Error()
			if tt.wantField != "" && !contains(errStr, tt.wantField) {
				t.Errorf("expected error to mention field %q, got: %s", tt.wantField, errStr)
			}
			if tt.wantCTE != "" && !contains(errStr, tt.wantCTE) {
				t.Errorf("expected error to mention CTE %q, got: %s", tt.wantCTE, errStr)
			}
		})
	}
}

func TestValidateQueryCTEs_ChainedCTEsValid(t *testing.T) {
	tables := testCatalog()

	sql := `WITH order_stats AS (
		SELECT user_id, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_spent
		FROM orders
		GROUP BY user_id
	),
	spending_percentiles AS (
		SELECT user_id, order_count, total_spent,
		       PERCENT_RANK() OVER (ORDER BY total_spent) as spend_percentile,
		       NTILE(4) OVER (ORDER BY total_spent) as spend_quartile
		FROM order_stats
	),
	tiered_users AS (
		SELECT sp.user_id, sp.order_count, sp.total_spent, sp.spend_percentile,
		       CASE WHEN sp.spend_quartile = 4 THEN 'platinum' ELSE 'bronze' END as tier
		FROM spending_percentiles sp
	)
	SELECT u.id, tiered_users.order_count, tiered_users.total_spent, tiered_users.tier
	FROM users u
	INNER JOIN tiered_users ON tiered_users.user_id = u.id`

	err := validateQueryCTEs(sql, "ChainedCTEs", tables)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateQueryCTEs_ChainedCTEsInvalid(t *testing.T) {
	tables := testCatalog()

	sql := `WITH order_stats AS (
		SELECT user_id, COUNT(*) as order_count
		FROM orders
		GROUP BY user_id
	),
	enriched AS (
		SELECT order_stats.user_id, order_stats.order_count, order_stats.total_spent
		FROM order_stats
	)
	SELECT * FROM enriched`

	err := validateQueryCTEs(sql, "ChainedCTEsInvalid", tables)
	if err == nil {
		t.Fatal("expected error for reference to nonexistent 'total_spent' in order_stats")
	}
	if !contains(err.Error(), "total_spent") {
		t.Errorf("expected error to mention 'total_spent', got: %s", err.Error())
	}
}

func TestExtractSelectListColumns(t *testing.T) {
	tests := []struct {
		body string
		want []string
	}{
		{
			body: "SELECT user_id, COUNT(*) as order_count FROM orders",
			want: []string{"user_id", "order_count"},
		},
		{
			body: "SELECT u.id as user_id, u.name, COALESCE(SUM(o.amount), 0) as total_spent FROM users u",
			want: []string{"user_id", "name", "total_spent"},
		},
		{
			body: "SELECT orders.user_id, COUNT(*) as order_count, COALESCE(SUM(orders.amount), 0) as total_spent, MAX(orders.created_at) as last_order_at FROM orders",
			want: []string{"user_id", "order_count", "total_spent", "last_order_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.body[:30], func(t *testing.T) {
			got := extractSelectListColumns(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d columns %v, want %d columns %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("column %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateCTEReferences_Integration(t *testing.T) {
	req := &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{
				{
					Name: "public",
					Tables: []*plugin.Table{
						{
							Rel: &plugin.Identifier{Name: "users"},
							Columns: []*plugin.Column{
								{Name: "id"},
								{Name: "name"},
								{Name: "email"},
								{Name: "created_at"},
							},
						},
						{
							Rel: &plugin.Identifier{Name: "orders"},
							Columns: []*plugin.Column{
								{Name: "id"},
								{Name: "user_id"},
								{Name: "amount"},
								{Name: "status"},
								{Name: "created_at"},
							},
						},
					},
				},
			},
		},
		Queries: []*plugin.Query{
			{
				Name: "GetValid",
				Cmd:  ":many",
				Text: `WITH order_totals AS (
					SELECT user_id, COUNT(*) as order_count
					FROM orders
					GROUP BY user_id
				)
				SELECT u.id, ot.order_count
				FROM users u
				INNER JOIN order_totals ot ON ot.user_id = u.id`,
			},
		},
	}

	if err := validateCTEReferences(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Now test with invalid reference
	req.Queries = []*plugin.Query{
		{
			Name: "GetInvalid",
			Cmd:  ":many",
			Text: `WITH order_totals AS (
				SELECT user_id, COUNT(*) as order_count
				FROM orders
				GROUP BY user_id
			)
			SELECT u.id, ot.order_count
			FROM users u
			INNER JOIN order_totals ot ON ot.user_id1 = u.id`,
		},
	}

	err := validateCTEReferences(req)
	if err == nil {
		t.Fatal("expected error for invalid CTE reference")
	}
	if !contains(err.Error(), "user_id1") {
		t.Errorf("expected error to mention 'user_id1', got: %s", err.Error())
	}
}

func TestValidateQueryCTEs_NestedSubquery(t *testing.T) {
	tables := testCatalog()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
		errCol  string
	}{
		{
			name: "CTE with nested subquery in SELECT - valid ref",
			sql: `WITH user_stats AS (
				SELECT user_id,
				       (SELECT COUNT(*) FROM orders o WHERE o.user_id = orders.user_id) as order_count
				FROM orders
				GROUP BY user_id
			)
			SELECT user_stats.user_id, user_stats.order_count FROM user_stats`,
			wantErr: false,
		},
		{
			name: "CTE with nested subquery in SELECT - invalid ref",
			sql: `WITH user_stats AS (
				SELECT user_id,
				       (SELECT COUNT(*) FROM orders o WHERE o.user_id = orders.user_id) as order_count
				FROM orders
				GROUP BY user_id
			)
			SELECT user_stats.user_id, user_stats.wrong_col FROM user_stats`,
			wantErr: true,
			errCol:  "wrong_col",
		},
		{
			name: "CTE with subquery in FROM",
			sql: `WITH totals AS (
				SELECT sq.user_id, sq.total
				FROM (SELECT user_id, SUM(amount) as total FROM orders GROUP BY user_id) sq
			)
			SELECT totals.user_id, totals.total FROM totals`,
			wantErr: false,
		},
		{
			name: "CTE with subquery in FROM - invalid ref",
			sql: `WITH totals AS (
				SELECT sq.user_id, sq.total
				FROM (SELECT user_id, SUM(amount) as total FROM orders GROUP BY user_id) sq
			)
			SELECT totals.user_id, totals.nonexistent FROM totals`,
			wantErr: true,
			errCol:  "nonexistent",
		},
		{
			name: "CTE with CASE containing subquery",
			sql: `WITH user_tiers AS (
				SELECT user_id,
				       CASE WHEN (SELECT COUNT(*) FROM orders o WHERE o.user_id = orders.user_id) > 5 THEN 'vip' ELSE 'regular' END as tier
				FROM orders
				GROUP BY user_id
			)
			SELECT user_tiers.user_id, user_tiers.tier FROM user_tiers`,
			wantErr: false,
		},
		{
			name: "CTE with EXISTS in WHERE - valid ref",
			sql: `WITH active_users AS (
				SELECT id, name
				FROM users u
				WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)
			)
			SELECT active_users.id, active_users.name FROM active_users`,
			wantErr: false,
		},
		{
			name: "CTE with EXISTS in WHERE - invalid ref",
			sql: `WITH active_users AS (
				SELECT id, name
				FROM users u
				WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)
			)
			SELECT active_users.id, active_users.wrong FROM active_users`,
			wantErr: true,
			errCol:  "wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryCTEs(tt.sql, tt.name, tables)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCol != "" && !contains(err.Error(), tt.errCol) {
					t.Errorf("expected error to mention %q, got: %s", tt.errCol, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateQueryCTEs_Union(t *testing.T) {
	tables := testCatalog()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
		errCol  string
	}{
		{
			name: "CTE with UNION ALL - valid ref",
			sql: `WITH all_names AS (
				SELECT id, name FROM users
				UNION ALL
				SELECT id, status as name FROM orders
			)
			SELECT all_names.id, all_names.name FROM all_names`,
			wantErr: false,
		},
		{
			name: "CTE with UNION ALL - invalid ref",
			sql: `WITH all_names AS (
				SELECT id, name FROM users
				UNION ALL
				SELECT id, status as name FROM orders
			)
			SELECT all_names.id, all_names.nonexistent FROM all_names`,
			wantErr: true,
			errCol:  "nonexistent",
		},
		{
			name: "CTE with UNION (no ALL) - valid ref",
			sql: `WITH combined AS (
				SELECT id, name FROM users WHERE id < 100
				UNION
				SELECT id, name FROM users WHERE id >= 100
			)
			SELECT combined.id, combined.name FROM combined`,
			wantErr: false,
		},
		{
			name: "CTE with UNION - invalid ref",
			sql: `WITH combined AS (
				SELECT id, name FROM users WHERE id < 100
				UNION
				SELECT id, name FROM users WHERE id >= 100
			)
			SELECT combined.id, combined.email FROM combined`,
			wantErr: true,
			errCol:  "email",
		},
		{
			name: "CTE with INTERSECT - valid ref",
			sql: `WITH common AS (
				SELECT user_id FROM orders WHERE amount > 100
				INTERSECT
				SELECT user_id FROM orders WHERE status = 'completed'
			)
			SELECT common.user_id FROM common`,
			wantErr: false,
		},
		{
			name: "CTE with EXCEPT - valid ref",
			sql: `WITH excluded AS (
				SELECT user_id FROM orders
				EXCEPT
				SELECT user_id FROM orders WHERE status = 'cancelled'
			)
			SELECT excluded.user_id FROM excluded`,
			wantErr: false,
		},
		{
			name: "CTE with multiple UNIONs - valid ref",
			sql: `WITH all_ids AS (
				SELECT id as entity_id, 'user' as entity_type FROM users
				UNION ALL
				SELECT id as entity_id, 'order' as entity_type FROM orders
			)
			SELECT all_ids.entity_id, all_ids.entity_type FROM all_ids`,
			wantErr: false,
		},
		{
			name: "CTE with multiple UNIONs - invalid ref",
			sql: `WITH all_ids AS (
				SELECT id as entity_id, 'user' as entity_type FROM users
				UNION ALL
				SELECT id as entity_id, 'order' as entity_type FROM orders
			)
			SELECT all_ids.entity_id, all_ids.wrong_type FROM all_ids`,
			wantErr: true,
			errCol:  "wrong_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryCTEs(tt.sql, tt.name, tables)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCol != "" && !contains(err.Error(), tt.errCol) {
					t.Errorf("expected error to mention %q, got: %s", tt.errCol, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateQueryCTEs_UnqualifiedReferences(t *testing.T) {
	tables := testCatalog()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
		errCol  string
	}{
		{
			name: "EXISTS with valid unqualified column",
			sql: `WITH active_users AS (
				SELECT id, name, email
				FROM users u
				WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)
			)
			SELECT active_users.id, active_users.name FROM active_users`,
			wantErr: false,
		},
		{
			name: "EXISTS with invalid unqualified column",
			sql: `WITH active_users AS (
				SELECT id, name, email
				FROM users u
				WHERE EXISTS (SELECT 1 FROM orders WHERE user_id1 = u.id)
			)
			SELECT active_users.id, active_users.name FROM active_users`,
			wantErr: true,
			errCol:  "user_id1",
		},
		{
			name: "subquery with valid unqualified WHERE",
			sql: `WITH stats AS (
				SELECT user_id,
				       (SELECT COUNT(*) FROM orders WHERE user_id = stats.user_id) as cnt
				FROM orders stats
				GROUP BY user_id
			)
			SELECT stats.user_id FROM stats`,
			wantErr: false,
		},
		{
			name: "subquery CASE with invalid unqualified column",
			sql: `WITH user_tiers AS (
				SELECT u.id as user_id, u.name,
				       CASE
				           WHEN (SELECT COUNT(*) FROM orders WHERE user_id1 = u.id) > 0 THEN 'active'
				           ELSE 'inactive'
				       END as status
				FROM users u
			)
			SELECT user_tiers.user_id, user_tiers.status FROM user_tiers`,
			wantErr: true,
			errCol:  "user_id1",
		},
		{
			name: "JOIN ON with valid unqualified column",
			sql: `WITH data AS (
				SELECT u.id as user_id, u.name
				FROM users u
				INNER JOIN orders ON user_id = u.id
			)
			SELECT data.user_id FROM data`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryCTEs(tt.sql, tt.name, tables)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCol != "" && !contains(err.Error(), tt.errCol) {
					t.Errorf("expected error to mention %q, got: %s", tt.errCol, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
