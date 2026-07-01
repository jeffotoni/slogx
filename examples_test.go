package slogx_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jeffotoni/slogx"
)

func ExampleNewCtx() {
	ctx, cancel := slogx.NewCtx().
		Set("X-Trace-ID", "abc-123").
		Set("X-User-ID", "user-42").
		Build()
	defer cancel()

	fmt.Println(slogx.CtxGet(ctx, "X-Trace-ID"))
	fmt.Println(slogx.CtxGet(ctx, "X-User-ID"))

	// Output:
	// abc-123
	// user-42
}

func ExampleWithCtx() {
	ctx := slogx.WithCtx(context.Background()).
		Any("attempt", 3).
		Bool("cached", true).
		Context()

	v, _ := slogx.CtxGetAny(ctx, "attempt")
	fmt.Println(v)

	// Output:
	// 3
}

func ExampleEntry_Ctx() {
	var buf bytes.Buffer
	log := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.INFO,
	})

	ctx := slogx.WithCtx(context.Background()).
		Any("attempt", 3).
		Context()

	log.Info().
		Ctx(ctx).
		Str("component", "auth").
		Msg("request").
		Send()

	out := buf.String()
	fmt.Println(
		strings.Contains(out, `"attempt":3`),
		strings.Contains(out, `"component":"auth"`),
		strings.Contains(out, `"msg":"request"`),
	)

	// Output:
	// true true true
}

func ExampleEntry_Number() {
	var buf bytes.Buffer
	log := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.INFO,
	})

	log.Info().
		Number("status", 200).
		Number("bytes", int64(1234)).
		Number("latency_ms", 12.3).
		Msg("ok").
		Send()

	out := buf.String()
	fmt.Println(
		strings.Contains(out, `"status":200`),
		strings.Contains(out, `"bytes":1234`),
		strings.Contains(out, `"latency_ms":12.3`),
	)

	// Output:
	// true true true
}

func ExampleEntry_Err() {
	var buf bytes.Buffer
	log := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.INFO,
	})

	log.Error().
		Err(errors.New("boom")).
		Msg("x").
		Send()

	out := buf.String()
	fmt.Println(strings.Contains(out, `"error":"boom"`))

	// Output:
	// true
}
