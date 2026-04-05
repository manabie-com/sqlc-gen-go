# sqlc-gen-go

A sqlc plugin that generates type-safe Go database access code from SQL. Runs as a WASM plugin (recommended) or standalone binary.

## Usage

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: https://github.com/vtuanjs/sqlc-gen-go/releases/download/v1.7.6/sqlc-gen-go.wasm
    sha256: d30e104014f568df2b1b4bcce6cabc8c49ba9d6dbba5c28e4b1a41729e3afc94
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      sql_package: pgx/v5
```

## Building from source

```sh
make all  # produces bin/sqlc-gen-go and bin/sqlc-gen-go.wasm
```

To use a local build:

```yaml
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""  # optional since sqlc v1.24.0
```

## Migrating from sqlc's built-in Go codegen

Two changes are required:

1. Add a top-level `plugins` entry pointing to the WASM plugin.
2. Replace `gen.go` with `codegen`, referencing the plugin by name. Move all options into the `options` block; `out` moves up one level.

**Before:**
```yaml
sql:
- engine: postgresql
  gen:
    go:
      package: db
      out: db
      emit_json_tags: true
```

**After:**
```yaml
plugins:
- name: golang
  wasm:
    url: https://github.com/vtuanjs/sqlc-gen-go/releases/download/v1.7.6/sqlc-gen-go.wasm
    sha256: d30e104014f568df2b1b4bcce6cabc8c49ba9d6dbba5c28e4b1a41729e3afc94
sql:
- engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      emit_json_tags: true
```

Global `overrides`/`go` move to `options`/`<plugin-name>`:

```yaml
options:
  golang:
    rename:
      id: "Identifier"
    overrides:
    - db_type: "timestamptz"
      nullable: true
      engine: postgresql
      go_type:
        import: "gopkg.in/guregu/null.v4"
        package: "null"
        type: "Time"
```

## Advanced Options

### `emit_per_file_queries`

Each SQL source file gets its own struct and interface instead of a shared `Queries`/`Querier`.

| SQL file | Struct | Interface |
|---|---|---|
| `users.sql` | `UsersQueries` | `UsersQuerier` |
| `user_orders.sql` | `UserOrdersQueries` | `UserOrdersQuerier` |

```yaml
options:
  emit_interface: true
  emit_per_file_queries: true
```

- Each `*.sql.go` contains its own struct, constructor, methods, and interface.
- `db.go` only keeps `DBTX`; `querier.go` is not generated.
- Incompatible with `emit_prepared_queries`.

---

### `emit_err_nil_if_no_rows`

`:one` queries return `nil, nil` instead of `nil, sql.ErrNoRows` when no row is found.

```yaml
options:
  emit_err_nil_if_no_rows: true
```

---

### `emit_tracing`

Injects custom code at the start of every query method. Supports `{{.MethodName}}` and `{{.StructName}}` template variables.

```yaml
options:
  emit_tracing:
    import: "go.opentelemetry.io/otel"
    package: "otel"
    code:
      - "ctx, span := otel.Tracer(\"{{.StructName}}\").Start(ctx, \"{{.MethodName}}\")"
      - "defer span.End()"
```

| Field | Description |
|---|---|
| `import` | Import path of the tracing package |
| `package` | Package alias (if different from the last path segment) |
| `code` | Lines to inject; each is a Go template |

---

### `emit_dynamic_filter`

Enables optional WHERE/ORDER BY clauses controlled at runtime via `-- :if @param` annotations in SQL.

When a parameter is marked with `:if`, the generated code:
- Makes the parameter a pointer (`*T`) in the params struct — `nil` means "skip this clause"
- Adds a `bool` field for flag-only parameters (e.g. ORDER BY toggles that appear only in `:if` annotations, not as `col = $N` predicate values)
- Calls the generated `DynamicSQL()` helper at runtime to strip inactive lines and renumber placeholders

```yaml
options:
  emit_dynamic_filter: true
```

**SQL annotations**

```sql
-- name: SearchUsers :many
SELECT * FROM users
WHERE
  1 = 1
  AND email = @email           -- :if @email        -- omit if email is nil
  AND phone = @phone           -- :if @phone        -- omit if phone is nil
  AND EXISTS (                 -- :if @has_orders   -- flag-only bool
    SELECT 1 FROM orders
    WHERE orders.user_id = users.id
      AND orders.created_at >= @orders_since  -- :if @orders_since
  )
ORDER BY id ASC;

-- name: SearchUsersOrdered :many
SELECT * FROM users
WHERE
  1 = 1
  AND email = @email -- :if @email
ORDER BY
  id ASC,  -- :if @id_asc
  id DESC  -- :if @id_desc
```

**Generated Go**

```go
type SearchUsersParams struct {
    Email       *string    // nil → clause skipped
    Phone       *string    // nil → clause skipped
    OrdersSince *time.Time // nil → clause skipped
    HasOrders   bool       // false → EXISTS block skipped
}

func (q *SearchQueries) SearchUsers(ctx context.Context, db DBTX, arg SearchUsersParams) ([]*User, error) {
...
    dynQuery, dynArgs := DynamicSQL(SearchUsers, []any{arg.Email, arg.Phone, arg.OrdersSince, arg.HasOrders})
    rows, err := db.Query(ctx, dynQuery, dynArgs...)
...
}
```

**Annotation rules**

| Syntax | Behaviour |
|---|---|
| `AND col = @param -- :if @param` | Inline — skip line if param is nil/false |
| `AND (a = @a AND b = @b) -- :if @a @b` | Multi-condition — skip if **any** param is inactive |
| `-- :if @flag` (standalone) | Block — skip the **next** line if flag is false |

A `DynamicSQL` helper is emitted into `dynfilter.go` in the output package. After filtering, remaining `$N` placeholders are renumbered sequentially and the args slice is trimmed to match, preventing "expected N arguments, got M" errors.

---

### `go_generate_mock`

Adds a `//go:generate` directive for mock generation. `$GOFILE` expands to the current filename at generate time.

- When `emit_per_file_queries` is enabled: the directive is added to each `*.sql.go` file.
- Otherwise: the directive is added only to the `querier.go` file.

```yaml
options:
  go_generate_mock: "mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock"
```

Running `go generate ./...` produces a mock per SQL file (with `emit_per_file_queries`):

| Source | Mock |
|---|---|
| `users.sql.go` | `mock/users.sql.go` |
| `orders.sql.go` | `mock/orders.sql.go` |
