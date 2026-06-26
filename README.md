# sqlc-gen-go

> Write SQL. Generate Go. Let the compiler be your AI's feedback loop.

sqlc-gen-go is a sqlc plugin designed as an **AI harness** — a tight, deterministic loop where AI writes SQL, the generator produces type-safe Go instantly, and the compiler catches mistakes before anything runs.

## Why this matters for AI-driven development

When AI writes database code, the most valuable thing you can give it is **fast, deterministic feedback**. sqlc-gen-go provides exactly that:

1. **Write SQL** — AI drafts queries directly against your schema
2. **`sqlc generate`** — produces type-safe Go structs and methods in milliseconds
3. **Compile** — the Go compiler catches type mismatches immediately
4. **Validate** — `enable_validate_cte` rejects invalid column references before code is generated
5. **Filter dynamically** — `emit_dynamic_filter` lets AI annotate SQL with `-- :if @param` to generate optional WHERE/ORDER BY clauses at runtime, no hand-written query builders needed
6. **Test** — `go_generate_mock` gives AI-written tests something to run against instantly
7. **Observe** — `emit_tracing` injects spans into every method automatically, no boilerplate

The loop is: **SQL → generate → compile → test → repeat.** No ORM guessing, no runtime surprises, no hand-written boilerplate for AI to get wrong.

Dynamic filtering is the feature that makes this harness practical for real-world queries. Instead of AI generating dozens of query variants or fragile string-concatenation builders, a single annotated SQL query handles every runtime combination — and the generated code is still fully type-safe.

## Usage

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: https://github.com/vtuanjs/sqlc-gen-go/releases/download/v3.1.0/sqlc-gen-go.wasm
    sha256: fd46f694ad7c9ff2dd13d6dab3291d2cc567e13c1aa63ba082c03aea4ed287d8
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
      emit_interface: true
      emit_json_tags: true
      emit_prepared_queries: true
      emit_per_file_queries: true
      emit_err_nil_if_no_rows: true
      emit_result_struct_pointers: true
      disable_result_slice_pointers: true
      emit_dynamic_filter: true
      enable_validate_cte: true
      go_generate_mock: "mockgen -source=$GOFILE -destination=mock/$GOFILE -package=mock"
      emit_tracing:
        import: "go.opentelemetry.io/otel"
        package: "otel"
        code:
          - 'ctx, span := otel.Tracer("{{.StructName}}").Start(ctx, "{{.MethodName}}")'
          - "defer span.End()"
```

> For all basic sqlc options (schema, queries, engine, etc.) see the [sqlc configuration reference](https://docs.sqlc.dev/en/stable/reference/config.html).

## Migrating from sqlc's built-in Go codegen

See [MIGRATION.md](MIGRATION.md) for step-by-step migration instructions.

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

Optional WHERE/ORDER BY clauses controlled at runtime via `-- :if @param` annotations. A `:if` parameter becomes a pointer (`*T`) in the params struct — `nil` skips its clause; flag-only params become `bool` fields.

```yaml
options:
  emit_dynamic_filter: true
```

```sql
-- name: SearchUsers :many
SELECT * FROM users
WHERE
  TRUE
  -- :if @phone
  AND phone = @phone
  -- :if @has_orders
  AND EXISTS (
    SELECT 1 FROM orders WHERE orders.user_id = users.id
  )
ORDER BY
  id ASC,   -- :if @id_asc
  id DESC,  -- :if @id_desc
  TRUE;
```

Use `TRUE` as the leading predicate (and trailing `ORDER BY` term) so the clause stays valid when lines are omitted.

All conditions skip when the referenced param is nil/false.

| Style | Syntax | Behaviour |
|---|---|---|
| Top-level | `-- :if @param` on its own line | Skip the **next** line |
| Top-level (multi-param) | `-- :if @a @b` on its own line | Skip the **next** line if **any** listed param is nil/false |
| Top-level Block `( )` | `-- :if @flag` then a line opening `(` | Skip the **entire block** (until matching `)`) |
| Inline | `expression -- :if @param` | Skip the expression (including trailing comma) |

---

### `disable_result_slice_pointers`

When `emit_result_struct_pointers: true` is set, `:many` queries return `[]*T` by default. Setting `disable_result_slice_pointers: true` keeps `:one` results as `*T` while changing `:many` results back to `[]T`.

Requires `emit_result_struct_pointers: true`.

```yaml
options:
  emit_result_struct_pointers: true
  disable_result_slice_pointers: true
```

| Query command | `emit_result_struct_pointers` only | + `disable_result_slice_pointers` |
|---|---|---|
| `:one` | `*MyRow` | `*MyRow` |
| `:many` | `[]*MyRow` | `[]MyRow` |

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

---

### `enable_validate_cte`

When `true`, validates column references across CTEs and tables before code generation. Each query is checked so that every `table.column` / unqualified column reference resolves to a real column in the relevant table or CTE; an invalid reference fails `sqlc generate` with an explanatory error instead of producing broken code.

Disabled by default (no validation overhead). Enable it per codegen output:

```yaml
options:
  enable_validate_cte: true
```

Example failure for a query referencing a non-existent `amount1` column:

```
query "GetExclusiveHighValueUsers": column "amount1" not found in any table in scope (orders)
```

---

## Known Issues

### Parameters in WHERE against CTE columns need an explicit type cast

When a query parameter (e.g. `@since`) is compared against a column that comes from a CTE (including `UNION ALL` CTEs), sqlc cannot infer the parameter's type and raises a confusing error:

```
table alias "<cte_name>" does not exist
```

**Fix:** cast the parameter to the target type at the call site:

```sql
-- Bad: sqlc cannot infer the type of @since
WITH all_entities AS (
    SELECT id, created_at FROM users
    UNION ALL
    SELECT id, created_at FROM orders
)
SELECT * FROM all_entities
WHERE all_entities.created_at >= @since;

-- Good: explicit cast tells sqlc the expected type
WITH all_entities AS (
    SELECT id, created_at FROM users
    UNION ALL
    SELECT id, created_at FROM orders
)
SELECT * FROM all_entities
WHERE all_entities.created_at >= @since::timestamp;
```

This is a core sqlc limitation — the plugin cannot improve the error message.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, how to run tests, and benchmarks.
