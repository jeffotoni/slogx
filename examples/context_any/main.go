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

	ctx := log.WithCtx(context.Background()).
		Any("attempt", 3).
		Bool("cached", true).
		Str("role", "admin").
		Context()

	attempt, _ := log.CtxGetAny(ctx, "attempt")
	cached, _ := log.CtxGetAny(ctx, "cached")
	role, _ := log.CtxGetAny(ctx, "role")

	fmt.Println("attempt:", attempt)
	fmt.Println("cached:", cached)
	fmt.Println("role:", role)

	if err := logger.Info().
		Ctx(ctx).
		Component("cache").
		Msg("typed context imported with WithCtx").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
