# Examples

## minimal

Minimal structured log without context.

```bash
go run ./examples/minimal
```

## context

Builds a `context.Context` with `NewCtx`, reads fields with `CtxGet`, and imports them into a log entry.

```bash
go run ./examples/context
```

## context_any

Adds typed fields to context with `WithCtx` and logs them through `Entry.Ctx(ctx)`.

```bash
go run ./examples/context_any
```

## context_precedence

Shows conflict resolution when the same key exists in `NewCtx`, `WithCtx` and the log entry itself.

```bash
go run ./examples/context_precedence
```

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

## trace_key

Customizes `TraceIDKey` and demonstrates trace propagation both directly on the entry and via context.

```bash
go run ./examples/trace_key
```

## error

Shows `Err(err)` with the default key and `Err("custom", err)` with a custom key.

```bash
go run ./examples/error
```

## typed_fields

Demonstrates numeric, boolean, duration and time helpers on one entry.

```bash
go run ./examples/typed_fields
```

## json_payload

Shows `JSON(...)` and `Any([]byte)` auto-detection for valid and invalid JSON bytes.

```bash
go run ./examples/json_payload
```

## caller

Adds the source file and line automatically with `Caller()`.

```bash
go run ./examples/caller
```

## level_filter

Demonstrates that entries below the configured minimum level are skipped.

```bash
go run ./examples/level_filter
```
