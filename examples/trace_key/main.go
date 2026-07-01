package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffotoni/log"
)

func main() {
	logger := log.New(log.Config{
		Format:      log.FormatText,
		Writer:      os.Stdout,
		Level:       log.INFO,
		ServiceName: "api",
		TraceIDKey:  "X-Trace-ID",
	})

	if err := logger.Info().
		TraceID("trace-direct-123").
		Msg("trace id set directly on entry").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	ctx, cancel := log.NewCtx(context.Background()).
		TraceKey("X-Trace-ID").
		TraceID("trace-context-456").
		Set("X-User-ID", "user-42").
		Build()
	defer cancel()

	ctx = log.WithCtx(ctx).
		TraceKey("X-Trace-ID").
		TraceID("trace-context-456").
		Context()

	if err := logger.Info().
		Ctx(ctx).
		Msg("trace id imported from context with custom key").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
