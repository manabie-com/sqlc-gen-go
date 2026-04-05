# DynamicSQL vs Manual benchmark

Compares `db.DynamicSQL` (generated, runtime SQL filtering) against equivalent
hand-written Go code (`manualSearchUsers`) for the `SearchUsers` query.

## Run

```sh
cd example
go test ./bench -bench=. -benchmem -count=3 -run='^$'
```

## Results (Apple M1 Pro)

| Benchmark                    | ns/op  | B/op | allocs/op |
|------------------------------|--------|------|-----------|
| DynamicSQL no optional       |  ~844  |  240 |     3     |
| Manual no optional           |  ~123  |  368 |     5     |
| DynamicSQL all optional      | ~1406  |  640 |     3     |
| Manual all optional          |  ~366  |  688 |     6     |

## Takeaways

- `DynamicSQL` is ~4-7x slower than manual code due to runtime line scanning
  and a per-call `map[int]int` allocation for placeholder remapping.
- `DynamicSQL` uses fewer allocations (3 vs 5-6) because it pre-grows a single
  `strings.Builder` buffer and avoids `fmt.Fprintf` per condition.
- A typical DB round-trip is ~1 ms; `DynamicSQL` overhead is ~1-1.5 us, negligible
  in practice. Use manual code only if query-building is on a hot path with no DB I/O.
