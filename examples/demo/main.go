package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffotoni/log"
)

func main() {
	ctx := log.WithCtx(context.Background()).
		TraceID("abc123").
		Str("X-User-ID", "user42").
		Any("attempt", 3).
		Context()

	fmt.Println("== json ==")
	if err := log.New(log.Config{
		Format:      log.FormatJSON,
		Writer:      os.Stdout,
		Level:       log.DEBUG,
		ServiceName: "api",
	}).
		Info().
		Ctx(ctx).
		Str("component", "auth").
		Msg("user login").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "json log error:", err)
	}

	fmt.Println("== text ==")
	if err := log.New(log.Config{
		Format:      log.FormatText,
		Writer:      os.Stdout,
		Level:       log.DEBUG,
		ServiceName: "api",
	}).
		Info().
		Ctx(ctx).
		Str("component", "auth").
		Msg("user login").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "text log error:", err)
	}

	fmt.Println("== slog ==")
	if err := log.New(log.Config{
		Format:      log.FormatSlog,
		Writer:      os.Stdout,
		Level:       log.DEBUG,
		ServiceName: "api",
	}).
		Info().
		Ctx(ctx).
		Str("component", "auth").
		Msg("user login").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "slog log error:", err)
	}
}
