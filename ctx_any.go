package slogx

import "context"

type anyFieldsKey struct{}

var internalAnyFieldsKey anyFieldsKey

func CtxGetAllAny(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}

	raw := ctx.Value(internalAnyFieldsKey)
	fields, ok := raw.(map[string]any)
	if !ok || len(fields) == 0 {
		return nil
	}

	out := make(map[string]any, len(fields))
	for k, v := range fields {
		out[k] = v
	}
	return out
}

func CtxGetAny(ctx context.Context, key string) (any, bool) {
	if ctx == nil || key == "" {
		return nil, false
	}

	raw := ctx.Value(internalAnyFieldsKey)
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}

	val, exists := fields[key]
	return val, exists
}

func CtxSetAllAny(ctx context.Context, fields map[string]any) context.Context {
	if ctx == nil {
		return nil
	}

	if fields == nil {
		return ctx
	}

	rawExisting := ctx.Value(internalAnyFieldsKey)
	existing, _ := rawExisting.(map[string]any)

	clone := make(map[string]any, len(fields)+len(existing))
	for k, v := range fields {
		if k == "" {
			continue
		}
		clone[k] = v
	}
	if len(clone) == 0 {
		return ctx
	}

	for k, v := range existing {
		if k == "" {
			continue
		}
		if _, ok := clone[k]; ok {
			continue
		}
		clone[k] = v
	}
	return context.WithValue(ctx, internalAnyFieldsKey, clone)
}

type CtxAnyBuilder struct {
	ctx      context.Context
	fields   map[string]any
	dirty    bool
	traceKey string
}

func WithCtx(ctx context.Context) *CtxAnyBuilder {
	return &CtxAnyBuilder{
		ctx:      ctx,
		traceKey: DefaultTraceIDKey,
	}
}

func (b *CtxAnyBuilder) TraceKey(key string) *CtxAnyBuilder {
	if b == nil {
		return b
	}
	if key != "" {
		b.traceKey = key
	}
	return b
}

func (b *CtxAnyBuilder) ensureFields() {
	if b == nil || b.ctx == nil || b.fields != nil {
		return
	}
	b.fields = CtxGetAllAny(b.ctx)
	if b.fields == nil {
		b.fields = make(map[string]any)
	}
}

func (b *CtxAnyBuilder) Any(key string, value any) *CtxAnyBuilder {
	if b == nil || b.ctx == nil || key == "" {
		return b
	}
	b.ensureFields()
	b.fields[key] = value
	b.dirty = true
	return b
}

func (b *CtxAnyBuilder) Str(key, value string) *CtxAnyBuilder {
	if value == "" {
		return b
	}
	return b.Any(key, value)
}

func (b *CtxAnyBuilder) Int(key string, value int) *CtxAnyBuilder { return b.Any(key, value) }

func (b *CtxAnyBuilder) Bool(key string, value bool) *CtxAnyBuilder { return b.Any(key, value) }

func (b *CtxAnyBuilder) TraceID(id string) *CtxAnyBuilder {
	if b == nil {
		return b
	}
	key := b.traceKey
	if key == "" {
		key = DefaultTraceIDKey
	}
	return b.Str(key, id)
}

func (b *CtxAnyBuilder) Context() context.Context {
	if b == nil {
		return nil
	}
	if !b.dirty {
		return b.ctx
	}
	b.ctx = CtxSetAllAny(b.ctx, b.fields)
	b.dirty = false
	return b.ctx
}
