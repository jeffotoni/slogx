# Examples

## demo

Shows the same log entry in 3 formats (`json`, `text`, `slog`).

```bash
go run ./examples/demo
```

## api

Complete HTTP flow example with:
- trace generation/propagation
- `NewCtx` + `WithCtx` context enrichment
- structured logs in request, validation, persistence and queue steps
- `Send()` error handling

Run:

```bash
go run ./examples/api
```

Test request:

```bash
curl -i -XPOST -H "Content-Type: application/json" localhost:8080/v1/user -d '{"name":"jeff","year":2026}'
```
