package slogx_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jeffotoni/slogx"
)

func TestNewCtx_Basic(t *testing.T) {
	ctx, cancel := slogx.NewCtx().Set("TraceID", "abc123").Build()
	defer cancel()

	if got := slogx.CtxGet(ctx, "TraceID"); got != "abc123" {
		t.Fatalf("expected TraceID=abc123, got %q", got)
	}
}

func TestNewCtx_CustomParentPreservesDeadline(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer parentCancel()

	ctx, cancel := slogx.NewCtx(parent).Set("k", "v").Build()
	defer cancel()

	if _, ok := ctx.Deadline(); !ok {
		t.Fatalf("expected deadline to be preserved from parent")
	}
}

func TestNewCtx_StringFieldsAreCumulative(t *testing.T) {
	ctx1, cancel1 := slogx.NewCtx().
		Set("a", "1").
		Build()
	defer cancel1()

	ctx2, cancel2 := slogx.NewCtx(ctx1).
		Set("b", "2").
		Build()
	defer cancel2()

	all := slogx.CtxGetAll(ctx2)
	if all["a"] != "1" {
		t.Fatalf("expected a=1, got %#v", all["a"])
	}
	if all["b"] != "2" {
		t.Fatalf("expected b=2, got %#v", all["b"])
	}
}

func TestWithCtx_TypedFields(t *testing.T) {
	ctx := slogx.WithCtx(context.Background()).
		Any("attempt", 3).
		TraceKey("MyTraceKey").
		TraceID("abc123").
		Context()

	if got := slogx.CtxGet(ctx, "MyTraceKey"); got != "abc123" {
		t.Fatalf("expected trace stored under MyTraceKey, got %q", got)
	}

	v, ok := slogx.CtxGetAny(ctx, "attempt")
	if !ok || v.(int) != 3 {
		t.Fatalf("expected attempt=3, got (%v, %v)", v, ok)
	}
}

func TestCtxSetAllAny_MergesExistingFields(t *testing.T) {
	ctx := slogx.CtxSetAllAny(context.Background(), map[string]any{"a": 1})
	ctx = slogx.CtxSetAllAny(ctx, map[string]any{"b": 2})

	all := slogx.CtxGetAllAny(ctx)
	if all["a"] != 1 {
		t.Fatalf("expected a=1, got %#v", all["a"])
	}
	if all["b"] != 2 {
		t.Fatalf("expected b=2, got %#v", all["b"])
	}
}

func TestWithCtx_TypedFieldsAreCumulative(t *testing.T) {
	ctx1 := slogx.WithCtx(context.Background()).Any("a", 1).Context()
	ctx2 := slogx.WithCtx(ctx1).Any("b", 2).Context()

	all := slogx.CtxGetAllAny(ctx2)
	if all["a"] != 1 {
		t.Fatalf("expected a=1, got %#v", all["a"])
	}
	if all["b"] != 2 {
		t.Fatalf("expected b=2, got %#v", all["b"])
	}
}

func TestCtxSetAllAny_NilFieldsDoesNotClearExisting(t *testing.T) {
	ctx := slogx.CtxSetAllAny(context.Background(), map[string]any{"a": 1})
	ctx = slogx.CtxSetAllAny(ctx, nil)

	all := slogx.CtxGetAllAny(ctx)
	if all["a"] != 1 {
		t.Fatalf("expected a=1 after nil update, got %#v", all["a"])
	}
}

func TestCtxSetAllAny_OnlyEmptyKeysDoesNotClearExisting(t *testing.T) {
	ctx := slogx.CtxSetAllAny(context.Background(), map[string]any{"a": 1})
	ctx = slogx.CtxSetAllAny(ctx, map[string]any{"": 9})

	all := slogx.CtxGetAllAny(ctx)
	if all["a"] != 1 {
		t.Fatalf("expected a=1 after empty-key update, got %#v", all["a"])
	}
	if _, ok := all[""]; ok {
		t.Fatalf("expected empty key to be ignored")
	}
}

func TestNewCtx_HighCardinalityKeys(t *testing.T) {
	builder := slogx.NewCtx()
	const total = 2000

	for i := 0; i < total; i++ {
		k := "k" + strconv.Itoa(i)
		v := "v" + strconv.Itoa(i)
		builder.Set(k, v)
	}

	ctx, cancel := builder.Build()
	defer cancel()

	for i := 0; i < total; i++ {
		k := "k" + strconv.Itoa(i)
		want := "v" + strconv.Itoa(i)
		if got := slogx.CtxGet(ctx, k); got != want {
			t.Fatalf("expected %s=%s, got %q", k, want, got)
		}
	}
}
