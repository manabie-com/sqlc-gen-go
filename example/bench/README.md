# DynamicSQL vs Manual benchmark

Compares `db.DynamicSQL` (generated, runtime SQL filtering) against equivalent
hand-written Go code for two query sizes:

- **Small**: `SearchUsers` — 1 required param + 4 optional conditions
- **Large**: 1 required param + 20 optional conditions

## Run

```sh
cd example
go test ./bench -bench=. -benchmem -count=3 -run='^$'
```

## Results (Apple M1 Pro)

### Small query (5 params, 4 optional)

| Benchmark                    | ns/op  |   req/s  | B/op | allocs/op |
|------------------------------|--------|----------|------|-----------|
| DynamicSQL no optional       |  ~839  |  ~1.19 M |  240 |     3     |
| Manual no optional           |  ~113  |  ~8.85 M |  368 |     5     |
| DynamicSQL all optional      | ~1321  |   ~757 K |  640 |     3     |
| Manual all optional          |  ~357  |  ~2.80 M |  688 |     6     |

### Large query (21 params, 20 optional)

| Benchmark                    | ns/op  |   req/s  |  B/op | allocs/op |
|------------------------------|--------|----------|-------|-----------|
| DynamicSQL no optional       | ~1820  |   ~549 K |   240 |     3     |
| Manual no optional           |  ~179  |  ~5.59 M |   640 |     5     |
| DynamicSQL all optional      | ~3736  |   ~268 K |  2632 |     9     |
| Manual all optional          | ~1842  |   ~543 K |  1920 |    27     |

## Takeaways

- `DynamicSQL` is ~7-10x slower than manual for the **no-optional** case because
  it scans every line regardless of how many conditions are active.
- For the **all-optional** case the gap narrows to ~2x on the large query (vs ~4x
  on the small query). Manual code calls `fmt.Fprintf` once per active condition,
  so its cost grows linearly; `DynamicSQL` has higher fixed overhead but lower
  per-condition cost.
- `DynamicSQL` allocates far fewer objects: 9 vs 27 allocs for 20 active conditions,
  because it avoids a `fmt.Fprintf` call per condition.
- A typical DB round-trip is ~1 ms; even the large-query `DynamicSQL` overhead is
  ~3.7 µs — negligible in practice. Prefer manual code only on hot paths with no
  DB I/O.
