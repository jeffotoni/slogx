package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jeffotoni/log"
)

func main() {
	logger := log.New(log.Config{
		Format:      log.FormatJSON,
		Writer:      os.Stdout,
		Level:       log.INFO,
		ServiceName: "api",
	})

	ctx, cancel := log.NewCtx(context.Background()).
		Set("X-Trace-ID", "abc-123").
		Set("X-User-ID", "user-42").
		Set("X-Session-ID", "sess-999").
		Timeout(5 * time.Second).
		Build()
	defer cancel()

	fmt.Println("trace:", log.CtxGet(ctx, "X-Trace-ID"))
	fmt.Println("user:", log.CtxGet(ctx, "X-User-ID"))
	fmt.Println("session:", log.CtxGet(ctx, "X-Session-ID"))

	if err := logger.Info().
		Ctx(ctx).
		Component("auth").
		Msg("context imported from NewCtx").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
