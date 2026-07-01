package main

import (
	"context"
	"fmt"
	"os"

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
		Set("role", "user").
		Set("tenant", "acme").
		Build()
	defer cancel()

	ctx = log.WithCtx(ctx).
		Str("role", "operator").
		Any("attempt", 1).
		Context()

	fmt.Println("expected role precedence: Entry > WithCtx > NewCtx")

	if err := logger.Info().
		Ctx(ctx).
		Str("role", "admin").
		Msg("per-entry fields override context defaults").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
