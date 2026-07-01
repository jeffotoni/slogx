package slogx

import (
	"context"
	"time"
)

type (
	ctxKey     string
	contextKey string
)

const internalKeysKey ctxKey = "slogx.internalKeys"

func getCtxKey(name string) contextKey {
	if name == "" {
		return contextKey("default")
	}
	return contextKey(name)
}

type CtxBuilder struct {
	parent     context.Context
	fields     map[string]string
	timeout    time.Duration
	useTimeout bool
	traceKey   string
}

func NewCtx(parent ...context.Context) *CtxBuilder {
	base := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		base = parent[0]
	}
	return &CtxBuilder{
		parent:   base,
		fields:   make(map[string]string),
		traceKey: DefaultTraceIDKey,
	}
}

func (b *CtxBuilder) TraceKey(key string) *CtxBuilder {
	if b == nil {
		return b
	}
	if key != "" {
		b.traceKey = key
	}
	return b
}

func (b *CtxBuilder) Set(key, value string) *CtxBuilder {
	if b == nil {
		return b
	}
	if key != "" && value != "" {
		b.fields[key] = value
	}
	return b
}

func (b *CtxBuilder) TraceID(id string) *CtxBuilder {
	if b == nil {
		return b
	}
	key := b.traceKey
	if key == "" {
		key = DefaultTraceIDKey
	}
	return b.Set(key, id)
}

func (b *CtxBuilder) Timeout(d time.Duration) *CtxBuilder {
	if b == nil {
		return b
	}
	if d > 0 {
		b.timeout = d
		b.useTimeout = true
	}
	return b
}

func (b *CtxBuilder) Background() *CtxBuilder {
	if b == nil {
		return b
	}
	b.useTimeout = false
	return b
}

func (b *CtxBuilder) Todo() *CtxBuilder { return b.Background() }

func (b CtxBuilder) Build() (context.Context, context.CancelFunc) {
	base := b.parent
	if base == nil {
		base = context.Background()
	}

	existingKeys, _ := base.Value(internalKeysKey).([]string)

	keys := make([]string, 0, len(existingKeys)+len(b.fields))
	seen := make(map[string]struct{}, len(existingKeys)+len(b.fields))
	for _, k := range existingKeys {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	for k, v := range b.fields {
		base = context.WithValue(base, getCtxKey(k), v)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	base = context.WithValue(base, internalKeysKey, keys)

	if b.useTimeout {
		return context.WithTimeout(base, b.timeout)
	}
	return context.WithCancel(base)
}

func CtxGet(ctx context.Context, key string) string {
	if ctx == nil || key == "" {
		return ""
	}

	if val, ok := ctx.Value(getCtxKey(key)).(string); ok {
		return val
	}

	if val, ok := CtxGetAny(ctx, key); ok {
		if s, ok := val.(string); ok {
			return s
		}
	}

	return ""
}

func CtxGetAll(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	rawKeys := ctx.Value(internalKeysKey)
	keyList, ok := rawKeys.([]string)
	if !ok || len(keyList) == 0 {
		return nil
	}

	out := make(map[string]string, len(keyList))
	for _, k := range keyList {
		val := ctx.Value(getCtxKey(k))
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}
