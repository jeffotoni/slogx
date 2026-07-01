package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffotoni/slogx"
)

func main() {
	ctx := slogx.WithCtx(context.Background()).
		TraceID("abc123").
		Str("X-User-ID", "user42").
		Any("attempt", 3).
		Context()

	fmt.Println("== json ==")
	if err := slogx.New(slogx.Config{
		Format:      slogx.FormatJSON,
		Writer:      os.Stdout,
		Level:       slogx.DEBUG,
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
	if err := slogx.New(slogx.Config{
		Format:      slogx.FormatText,
		Writer:      os.Stdout,
		Level:       slogx.DEBUG,
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
	if err := slogx.New(slogx.Config{
		Format:      slogx.FormatSlog,
		Writer:      os.Stdout,
		Level:       slogx.DEBUG,
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
