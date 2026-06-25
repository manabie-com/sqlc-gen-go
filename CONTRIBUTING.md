# Contributing

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

## Testing

```sh
make test          # plugin internals + generated-code unit tests (no DB)
make example-e2e   # end-to-end tests against a real PostgreSQL database
```

Run a single test:

```sh
go test ./internal/... -run TestName
```

Run fuzz tests:

```sh
go test ./internal/opts/... -fuzz FuzzOverride
```

## Benchmarks

`emit_dynamic_filter` uses a pre-compiled query (parsed once at init, `Build()` per request) rather than parsing on every call. It is ~14–25× faster than one-shot parsing and matches or beats hand-written builders, with allocations down to 3–10 per call. Source and full numbers: [`example/bench/`](example/bench/).

```sh
cd example && go test ./bench -bench=. -benchmem -run='^$'
```
