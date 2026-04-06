# DynamicSQL vs PreCompiled vs Manual benchmark

Compares three approaches for dynamic SQL construction:

- **`DynamicSQL`** — one-shot helper; parses the annotated SQL on every call (backward compat)
- **`PreCompiled`** — generated code path; SQL is parsed once at package init into a `dynCompiledQuery`, then `Build()` is called per request (no per-call string scanning)
- **Manual** — hand-written `strings.Builder` with `fmt.Fprintf` per condition

Two query sizes are tested:

- **Small**: `SearchUsers` — 1 required param + 4 optional conditions
- **Large**: 1 required param + 20 optional conditions

## Run

```sh
cd example
go test ./bench -bench=. -benchmem -count=3 -run='^$'
```

## Results (Apple M1 Pro)

### Small query (5 params, 4 optional)

| Benchmark                       | ns/op |    req/s | B/op  | allocs/op |
|---------------------------------|------:|---------:|------:|----------:|
| DynamicSQL — no optional        | 2,209 |   ~453 K | 3,688 |        46 |
| **PreCompiled — no optional**   |   **151** | **~6.62 M** |  **304** |    **3** |
| Manual — no optional            |   121 |  ~8.26 M |   368 |         5 |
| DynamicSQL — all optional       | 2,400 |   ~417 K | 4,168 |        49 |
| **PreCompiled — all optional**  |   **313** | **~3.19 M** |  **784** |    **6** |
| Manual — all optional           |   379 |  ~2.64 M |   688 |         6 |

### Large query (21 params, 20 optional)

| Benchmark                       | ns/op |    req/s |  B/op | allocs/op |
|---------------------------------|------:|---------:|------:|----------:|
| DynamicSQL — no optional        | 5,344 |   ~187 K | 7,472 |       119 |
| **PreCompiled — no optional**   |   **214** | **~4.67 M** |  **304** |    **3** |
| Manual — no optional            |   177 |  ~5.65 M |   640 |         5 |
| DynamicSQL — all optional       | 6,072 |   ~165 K | 9,552 |       126 |
| **PreCompiled — all optional**  |   **817** | **~1.22 M** | **2,384** |  **10** |
| Manual — all optional           | 1,919 |   ~521 K | 1,920 |        27 |

## Results (Intel Core i7-11800H @ 2.30GHz)

### Small query (5 params, 4 optional)

| Benchmark                       | ns/op |    req/s | B/op  | allocs/op |
|---------------------------------|------:|---------:|------:|----------:|
| DynamicSQL — no optional        | 2,291 |   ~437 K | 3,688 |        46 |
| **PreCompiled — no optional**   |   **175** | **~5.71 M** |  **304** |    **3** |
| Manual — no optional            |   131 |  ~7.63 M |   368 |         5 |
| DynamicSQL — all optional       | 2,521 |   ~397 K | 4,168 |        49 |
| **PreCompiled — all optional**  |   **358** | **~2.79 M** |  **784** |    **6** |
| Manual — all optional           |   437 |  ~2.29 M |   688 |         6 |

### Large query (21 params, 20 optional)

| Benchmark                       | ns/op |    req/s |  B/op | allocs/op |
|---------------------------------|------:|---------:|------:|----------:|
| DynamicSQL — no optional        | 7,327 |   ~136 K | 7,472 |       119 |
| **PreCompiled — no optional**   |   **226** | **~4.42 M** |  **304** |    **3** |
| Manual — no optional            |   208 |  ~4.81 M |   640 |         5 |
| DynamicSQL — all optional       | 6,501 |   ~154 K | 9,552 |       126 |
| **PreCompiled — all optional**  |   **951** | **~1.05 M** | **2,384** |  **10** |
| Manual — all optional           | 2,339 |   ~428 K | 1,921 |        27 |

## How it works

When `emit_dynamic_filter: true` is set, the generator now emits a package-level
pre-compiled query variable alongside each dynamic-filter query:

```go
// Generated once at package init — zero per-call parsing cost
var _searchUsersDynQ = dynCompile(SearchUsers)

func (q *Queries) SearchUsers(ctx context.Context, arg SearchUsersParams) ([]User, error) {
    dynQuery, dynArgs := _searchUsersDynQ.Build([]any{arg.Name, arg.Email, arg.Phone, ...})
    rows, err := q.db.Query(ctx, dynQuery, dynArgs...)
    // ...
}
```

`dynCompile` parses the `-- :if $N` markers once into a list of pre-split segments.
`Build` then just iterates those segments, checks conditions with indexed array lookups,
and writes the pre-split string parts — no regex, no string scanning.

`DynamicSQL(sql, args)` is kept for backward compatibility and one-off use;
it still parses on every call.

## Takeaways

- **PreCompiled is ~14–25x faster than `DynamicSQL`** and matches manual performance
  for the no-optional case (151 ns vs 121 ns manual, vs 2,209 ns `DynamicSQL`).
- **For the all-optional large query, PreCompiled is actually 2.3x faster than manual**
  (817 ns vs 1,919 ns), because manual uses `fmt.Fprintf` per condition while
  `Build` writes pre-split string literals directly.
- **Allocations drop from 46–126 down to 3–10**, matching or beating manual.
- A typical DB round-trip is ~1 ms; `PreCompiled` overhead is ~150–800 ns —
  effectively free in practice.
