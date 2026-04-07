package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func TestPostgresType(t *testing.T) {
	base := makeReqWithCatalog()

	pgv5 := func(typ string, notNull bool) string {
		return postgresType(base, &opts.Options{SqlPackage: "pgx/v5"}, makeCol(typ, notNull))
	}
	pgv4 := func(typ string, notNull bool) string {
		return postgresType(base, &opts.Options{SqlPackage: "pgx/v4"}, makeCol(typ, notNull))
	}
	sql := func(typ string, notNull bool) string {
		return postgresType(base, &opts.Options{SqlPackage: "database/sql"}, makeCol(typ, notNull))
	}
	ptr := func(typ string, notNull bool) string {
		return postgresType(base, &opts.Options{SqlPackage: "pgx/v5", EmitPointersForNullTypes: true}, makeCol(typ, notNull))
	}

	t.Run("integers_not_null", func(t *testing.T) {
		cases := []struct{ typ, want string }{
			{"serial", "int32"}, {"serial4", "int32"}, {"pg_catalog.serial4", "int32"},
			{"bigserial", "int64"}, {"serial8", "int64"}, {"pg_catalog.serial8", "int64"},
			{"smallserial", "int16"}, {"serial2", "int16"}, {"pg_catalog.serial2", "int16"},
			{"integer", "int32"}, {"int", "int32"}, {"int4", "int32"}, {"pg_catalog.int4", "int32"},
			{"bigint", "int64"}, {"int8", "int64"}, {"pg_catalog.int8", "int64"},
			{"smallint", "int16"}, {"int2", "int16"}, {"pg_catalog.int2", "int16"},
		}
		for _, tc := range cases {
			if got := pgv5(tc.typ, true); got != tc.want {
				t.Errorf("pgv5(%q, true) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("integers_nullable_pgxv5", func(t *testing.T) {
		cases := []struct{ typ, want string }{
			{"serial", "pgtype.Int4"}, {"bigserial", "pgtype.Int8"}, {"smallserial", "pgtype.Int2"},
			{"integer", "pgtype.Int4"}, {"bigint", "pgtype.Int8"}, {"smallint", "pgtype.Int2"},
		}
		for _, tc := range cases {
			if got := pgv5(tc.typ, false); got != tc.want {
				t.Errorf("pgv5(%q, false) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("integers_nullable_pgxv4_and_stdlib", func(t *testing.T) {
		cases := []struct {
			fn   func(string, bool) string
			typ  string
			want string
		}{
			{pgv4, "serial", "sql.NullInt32"},
			{pgv4, "bigserial", "sql.NullInt64"},
			{pgv4, "smallserial", "sql.NullInt16"},
			{sql, "integer", "sql.NullInt32"},
			{sql, "bigint", "sql.NullInt64"},
			{sql, "smallint", "sql.NullInt16"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, false); got != tc.want {
				t.Errorf("nullable(%q) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("floats", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "float", true, "float64"},
			{pgv5, "double precision", true, "float64"},
			{pgv5, "float8", true, "float64"},
			{pgv5, "float", false, "pgtype.Float8"},
			{sql, "float", false, "sql.NullFloat64"},
			{pgv5, "real", true, "float32"},
			{pgv5, "float4", true, "float32"},
			{pgv5, "real", false, "pgtype.Float4"},
			{sql, "real", false, "sql.NullFloat64"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("numeric_and_money", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "numeric", true, "pgtype.Numeric"},
			{pgv5, "numeric", false, "pgtype.Numeric"},
			{pgv4, "numeric", true, "pgtype.Numeric"},
			{sql, "numeric", true, "string"},
			{sql, "numeric", false, "sql.NullString"},
			{sql, "money", true, "string"},
			{pgv5, "pg_catalog.numeric", true, "pgtype.Numeric"},
			{pgv5, "money", true, "pgtype.Numeric"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("boolean", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			notNull bool
			want    string
		}{
			{pgv5, true, "bool"},
			{pgv5, false, "pgtype.Bool"},
			{pgv4, false, "sql.NullBool"},
			{sql, true, "bool"},
			{sql, false, "sql.NullBool"},
		}
		for _, tc := range cases {
			if got := tc.fn("boolean", tc.notNull); got != tc.want {
				t.Errorf("boolean(%v) = %q, want %q", tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("text_variants", func(t *testing.T) {
		types := []string{"text", "pg_catalog.varchar", "pg_catalog.bpchar", "string", "citext", "name"}
		for _, typ := range types {
			if got := pgv5(typ, true); got != "string" {
				t.Errorf("pgv5(%q, true) = %q, want string", typ, got)
			}
			if got := pgv5(typ, false); got != "pgtype.Text" {
				t.Errorf("pgv5(%q, false) = %q, want pgtype.Text", typ, got)
			}
			if got := sql(typ, false); got != "sql.NullString" {
				t.Errorf("sql(%q, false) = %q, want sql.NullString", typ, got)
			}
		}
	})

	t.Run("json_jsonb", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "json", true, "[]byte"},
			{pgv5, "json", false, "[]byte"},
			{pgv4, "json", true, "pgtype.JSON"},
			{sql, "json", true, "json.RawMessage"},
			{sql, "json", false, "pqtype.NullRawMessage"},
			{pgv5, "jsonb", true, "[]byte"},
			{pgv4, "jsonb", true, "pgtype.JSONB"},
			{sql, "jsonb", true, "json.RawMessage"},
			{sql, "jsonb", false, "pqtype.NullRawMessage"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("bytea_blob", func(t *testing.T) {
		for _, typ := range []string{"bytea", "blob", "pg_catalog.bytea"} {
			if got := pgv5(typ, true); got != "[]byte" {
				t.Errorf("(%q) = %q, want []byte", typ, got)
			}
			if got := pgv5(typ, false); got != "[]byte" {
				t.Errorf("(%q, false) = %q, want []byte", typ, got)
			}
		}
	})

	t.Run("uuid", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			notNull bool
			want    string
		}{
			{pgv5, true, "pgtype.UUID"},
			{pgv5, false, "pgtype.UUID"},
			{pgv4, true, "uuid.UUID"},
			{pgv4, false, "uuid.NullUUID"},
			{sql, true, "uuid.UUID"},
			{sql, false, "uuid.NullUUID"},
		}
		for _, tc := range cases {
			if got := tc.fn("uuid", tc.notNull); got != tc.want {
				t.Errorf("uuid(%v) = %q, want %q", tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("date_time_types", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "date", true, "pgtype.Date"},
			{pgv5, "date", false, "pgtype.Date"},
			{pgv4, "date", true, "time.Time"},
			{pgv4, "date", false, "sql.NullTime"},
			{sql, "date", true, "time.Time"},
			{sql, "date", false, "sql.NullTime"},
			{pgv5, "pg_catalog.time", true, "pgtype.Time"},
			{pgv5, "pg_catalog.time", false, "pgtype.Time"},
			{pgv4, "pg_catalog.time", true, "time.Time"},
			{pgv4, "pg_catalog.time", false, "sql.NullTime"},
			{pgv5, "pg_catalog.timetz", true, "time.Time"},
			{pgv5, "pg_catalog.timetz", false, "sql.NullTime"},
			{pgv5, "pg_catalog.timestamp", true, "pgtype.Timestamp"},
			{pgv5, "pg_catalog.timestamp", false, "pgtype.Timestamp"},
			{pgv4, "pg_catalog.timestamp", true, "time.Time"},
			{pgv4, "pg_catalog.timestamp", false, "sql.NullTime"},
			{pgv5, "pg_catalog.timestamptz", true, "pgtype.Timestamptz"},
			{pgv5, "pg_catalog.timestamptz", false, "pgtype.Timestamptz"},
			{pgv5, "timestamptz", true, "pgtype.Timestamptz"},
			{sql, "timestamptz", true, "time.Time"},
			{sql, "timestamptz", false, "sql.NullTime"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("inet_cidr_macaddr", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "inet", true, "netip.Addr"},
			{pgv5, "inet", false, "*netip.Addr"},
			{pgv4, "inet", true, "pgtype.Inet"},
			{sql, "inet", true, "pqtype.Inet"},
			{pgv5, "cidr", true, "netip.Prefix"},
			{pgv5, "cidr", false, "*netip.Prefix"},
			{pgv4, "cidr", true, "pgtype.CIDR"},
			{sql, "cidr", true, "pqtype.CIDR"},
			{pgv5, "macaddr", true, "net.HardwareAddr"},
			{pgv4, "macaddr", true, "pgtype.Macaddr"},
			{sql, "macaddr", true, "pqtype.Macaddr"},
			{pgv5, "macaddr8", true, "net.HardwareAddr"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("ltree_variants", func(t *testing.T) {
		for _, typ := range []string{"ltree", "lquery", "ltxtquery"} {
			if got := pgv5(typ, true); got != "string" {
				t.Errorf("pgv5(%q, true) = %q, want string", typ, got)
			}
			if got := pgv5(typ, false); got != "pgtype.Text" {
				t.Errorf("pgv5(%q, false) = %q, want pgtype.Text", typ, got)
			}
			if got := sql(typ, false); got != "sql.NullString" {
				t.Errorf("sql(%q, false) = %q, want sql.NullString", typ, got)
			}
		}
	})

	t.Run("interval", func(t *testing.T) {
		cases := []struct {
			fn      func(string, bool) string
			typ     string
			notNull bool
			want    string
		}{
			{pgv5, "interval", true, "pgtype.Interval"},
			{pgv5, "interval", false, "pgtype.Interval"},
			{pgv4, "interval", true, "int64"},
			{pgv4, "interval", false, "sql.NullInt64"},
			{sql, "pg_catalog.interval", true, "int64"},
			{sql, "pg_catalog.interval", false, "sql.NullInt64"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, tc.notNull); got != tc.want {
				t.Errorf("(%q, %v) = %q, want %q", tc.typ, tc.notNull, got, tc.want)
			}
		}
	})

	t.Run("ranges", func(t *testing.T) {
		cases := []struct {
			fn   func(string, bool) string
			typ  string
			want string
		}{
			{pgv4, "daterange", "pgtype.Daterange"},
			{pgv5, "daterange", "pgtype.Range[pgtype.Date]"},
			{sql, "daterange", "interface{}"},
			{pgv5, "datemultirange", "pgtype.Multirange[pgtype.Range[pgtype.Date]]"},
			{sql, "datemultirange", "interface{}"},
			{pgv4, "tsrange", "pgtype.Tsrange"},
			{pgv5, "tsrange", "pgtype.Range[pgtype.Timestamp]"},
			{sql, "tsrange", "interface{}"},
			{pgv5, "tsmultirange", "pgtype.Multirange[pgtype.Range[pgtype.Timestamp]]"},
			{pgv4, "tstzrange", "pgtype.Tstzrange"},
			{pgv5, "tstzrange", "pgtype.Range[pgtype.Timestamptz]"},
			{pgv5, "tstzmultirange", "pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]"},
			{pgv4, "numrange", "pgtype.Numrange"},
			{pgv5, "numrange", "pgtype.Range[pgtype.Numeric]"},
			{pgv5, "nummultirange", "pgtype.Multirange[pgtype.Range[pgtype.Numeric]]"},
			{pgv4, "int4range", "pgtype.Int4range"},
			{pgv5, "int4range", "pgtype.Range[pgtype.Int4]"},
			{pgv5, "int4multirange", "pgtype.Multirange[pgtype.Range[pgtype.Int4]]"},
			{pgv4, "int8range", "pgtype.Int8range"},
			{pgv5, "int8range", "pgtype.Range[pgtype.Int8]"},
			{pgv5, "int8multirange", "pgtype.Multirange[pgtype.Range[pgtype.Int8]]"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, true); got != tc.want {
				t.Errorf("(%q) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("hstore", func(t *testing.T) {
		if got := pgv5("hstore", true); got != "pgtype.Hstore" {
			t.Errorf("pgv5 hstore = %q, want pgtype.Hstore", got)
		}
		if got := pgv4("hstore", true); got != "pgtype.Hstore" {
			t.Errorf("pgv4 hstore = %q, want pgtype.Hstore", got)
		}
		if got := sql("hstore", true); got != "interface{}" {
			t.Errorf("sql hstore = %q, want interface{}", got)
		}
	})

	t.Run("bit_varbit", func(t *testing.T) {
		cases := []struct {
			fn   func(string, bool) string
			typ  string
			want string
		}{
			{pgv5, "bit", "pgtype.Bits"},
			{pgv4, "bit", "pgtype.Varbit"},
			{pgv5, "varbit", "pgtype.Bits"},
			{pgv4, "varbit", "pgtype.Varbit"},
			{pgv5, "pg_catalog.bit", "pgtype.Bits"},
			{pgv5, "pg_catalog.varbit", "pgtype.Bits"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, true); got != tc.want {
				t.Errorf("(%q) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("system_oid_types", func(t *testing.T) {
		cases := []struct {
			fn   func(string, bool) string
			typ  string
			want string
		}{
			{pgv5, "cid", "pgtype.Uint32"},
			{pgv4, "cid", "pgtype.CID"},
			{pgv5, "oid", "pgtype.Uint32"},
			{pgv4, "oid", "pgtype.OID"},
			{pgv5, "tid", "pgtype.TID"},
			{pgv4, "tid", "pgtype.TID"},
			{pgv5, "xid", "pgtype.Uint32"},
			{pgv4, "xid", "pgtype.XID"},
		}
		for _, tc := range cases {
			if got := tc.fn(tc.typ, true); got != tc.want {
				t.Errorf("(%q) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("geometric_types", func(t *testing.T) {
		types := []struct{ typ, want string }{
			{"box", "pgtype.Box"}, {"circle", "pgtype.Circle"},
			{"line", "pgtype.Line"}, {"lseg", "pgtype.Lseg"},
			{"path", "pgtype.Path"}, {"point", "pgtype.Point"},
			{"polygon", "pgtype.Polygon"},
		}
		for _, tc := range types {
			for _, fn := range []func(string, bool) string{pgv4, pgv5} {
				if got := fn(tc.typ, true); got != tc.want {
					t.Errorf("(%q) = %q, want %q", tc.typ, got, tc.want)
				}
			}
		}
	})

	t.Run("vector", func(t *testing.T) {
		if got := pgv5("vector", true); got != "pgvector.Vector" {
			t.Errorf("pgv5 vector notNull = %q, want pgvector.Vector", got)
		}
		if got := pgv5("vector", false); got != "pgvector.Vector" {
			t.Errorf("pgv5 vector nullable = %q, want pgvector.Vector", got)
		}
		if got := ptr("vector", false); got != "*pgvector.Vector" {
			t.Errorf("ptr vector nullable = %q, want *pgvector.Vector", got)
		}
		if got := sql("vector", true); got != "interface{}" {
			t.Errorf("sql vector = %q, want interface{}", got)
		}
	})

	t.Run("void_and_any", func(t *testing.T) {
		if got := pgv5("void", true); got != "interface{}" {
			t.Errorf("void = %q, want interface{}", got)
		}
		if got := pgv5("any", true); got != "interface{}" {
			t.Errorf("any = %q, want interface{}", got)
		}
	})

	t.Run("emit_pointers_for_null", func(t *testing.T) {
		cases := []struct{ typ, want string }{
			{"integer", "*int32"},
			{"bigint", "*int64"},
			{"smallint", "*int16"},
			{"serial", "*int32"},
			{"float", "*float64"},
			{"real", "*float32"},
			{"boolean", "*bool"},
			{"text", "*string"},
			{"ltree", "*string"},
		}
		for _, tc := range cases {
			if got := ptr(tc.typ, false); got != tc.want {
				t.Errorf("ptr(%q, false) = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("enum_from_catalog", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog: &plugin.Catalog{
				DefaultSchema: "public",
				Schemas: []*plugin.Schema{
					{
						Name:  "public",
						Enums: []*plugin.Enum{{Name: "status", Vals: []string{"active", "inactive"}}},
					},
				},
			},
			Settings: &plugin.Settings{Engine: "postgresql"},
		}
		o := &opts.Options{SqlPackage: "pgx/v5", InitialismsMap: map[string]struct{}{"id": {}}}
		if got := postgresType(req, o, makeCol("status", true)); got != "Status" {
			t.Errorf("enum notNull = %q, want Status", got)
		}
		if got := postgresType(req, o, makeCol("status", false)); got != "NullStatus" {
			t.Errorf("enum nullable = %q, want NullStatus", got)
		}
	})

	t.Run("enum_non_default_schema", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog: &plugin.Catalog{
				DefaultSchema: "public",
				Schemas: []*plugin.Schema{
					{
						Name:  "myschema",
						Enums: []*plugin.Enum{{Name: "role", Vals: []string{"admin", "user"}}},
					},
				},
			},
			Settings: &plugin.Settings{Engine: "postgresql"},
		}
		o := &opts.Options{SqlPackage: "pgx/v5", InitialismsMap: map[string]struct{}{"id": {}}}
		if got := postgresType(req, o, makeCol("myschema.role", true)); got != "MyschemaRole" {
			t.Errorf("enum non-default schema notNull = %q, want MyschemaRole", got)
		}
		if got := postgresType(req, o, makeCol("myschema.role", false)); got != "NullMyschemaRole" {
			t.Errorf("enum non-default schema nullable = %q, want NullMyschemaRole", got)
		}
	})

	t.Run("composite_type_from_catalog", func(t *testing.T) {
		req := &plugin.GenerateRequest{
			Catalog: &plugin.Catalog{
				DefaultSchema: "public",
				Schemas: []*plugin.Schema{
					{
						Name:           "public",
						CompositeTypes: []*plugin.CompositeType{{Name: "address"}},
					},
				},
			},
			Settings: &plugin.Settings{Engine: "postgresql"},
		}
		o := &opts.Options{SqlPackage: "pgx/v5"}
		if got := postgresType(req, o, makeCol("address", true)); got != "string" {
			t.Errorf("composite notNull = %q, want string", got)
		}
		if got := postgresType(req, o, makeCol("address", false)); got != "sql.NullString" {
			t.Errorf("composite nullable = %q, want sql.NullString", got)
		}
		oPtr := &opts.Options{SqlPackage: "pgx/v5", EmitPointersForNullTypes: true}
		if got := postgresType(req, oPtr, makeCol("address", false)); got != "*string" {
			t.Errorf("composite nullable+emitPointers = %q, want *string", got)
		}
	})

	t.Run("unknown_type_falls_through", func(t *testing.T) {
		if got := pgv5("totally_unknown_xyz", true); got != "interface{}" {
			t.Errorf("unknown type = %q, want interface{}", got)
		}
	})
}
