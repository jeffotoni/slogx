package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const maxTextBufCap = 64 * 1024

type textBuf struct {
	b []byte
}

func trimSpaceFast(s string) string {
	if s == "" {
		return s
	}
	b0 := s[0]
	b1 := s[len(s)-1]
	if b0 < utf8.RuneSelf && b1 < utf8.RuneSelf && b0 > ' ' && b1 > ' ' {
		return s
	}
	return strings.TrimSpace(s)
}

func replaceBytes(buf []byte, pos int, oldLen int, s string) []byte {
	if oldLen < 0 || pos < 0 || pos+oldLen > len(buf) {
		return buf
	}

	newLen := len(s)
	diff := newLen - oldLen
	if diff == 0 {
		copy(buf[pos:pos+newLen], s)
		return buf
	}

	origLen := len(buf)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			buf = append(buf, 0)
		}
		copy(buf[pos+newLen:], buf[pos+oldLen:origLen])
	} else {
		copy(buf[pos+newLen:], buf[pos+oldLen:origLen])
		buf = buf[:origLen+diff]
	}
	copy(buf[pos:pos+newLen], s)
	return buf
}

var textBufPool = sync.Pool{
	New: func() any {
		return &textBuf{b: make([]byte, 0, 256)}
	},
}

type traceKeySet struct {
	k0 string
	k1 string
	k2 string
}

func newTraceKeySet(primary string) traceKeySet {
	var t traceKeySet
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if k == t.k0 || k == t.k1 || k == t.k2 {
			return
		}
		switch {
		case t.k0 == "":
			t.k0 = k
		case t.k1 == "":
			t.k1 = k
		case t.k2 == "":
			t.k2 = k
		}
	}

	add(primary)
	add(DefaultTraceIDKey)
	add("trace_id")
	return t
}

type textHandler struct {
	w          io.Writer
	minLevel   slog.Level
	timeFormat string
	sep        string
	traceKeys  traceKeySet

	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func newTextHandler(cfg Config, _ *slog.HandlerOptions) slog.Handler {
	return &textHandler{
		w:          cfg.Writer,
		minLevel:   toSlogLevel(cfg.Level),
		timeFormat: cfg.TimeFormat,
		sep:        cfg.Separator,
		traceKeys:  newTraceKeySet(cfg.TraceIDKey),
		mu:         &sync.Mutex{},
	}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(nil, r.Level) {
		return nil
	}

	tb := textBufPool.Get().(*textBuf)
	buf := tb.b[:0]

	if !r.Time.IsZero() {
		buf = r.Time.AppendFormat(buf, h.timeFormat)
	}

	msg := trimSpaceFast(r.Message)
	if msg == "" {
		msg = "-"
	}

	buf = append(buf, h.sep...)
	buf = append(buf, levelString(r.Level)...)
	buf = append(buf, h.sep...)
	tracePos := len(buf)
	buf = append(buf, '-')
	buf = append(buf, h.sep...)
	buf = append(buf, msg...)

	for _, a := range h.attrs {
		if h.isTraceKey(a.Key) {
			if tracePos >= 0 {
				traceID := h.traceValueFromAttr(a)
				if traceID != "" && traceID != "-" {
					buf = replaceBytes(buf, tracePos, 1, traceID)
				}
				tracePos = -1
			}
			continue
		}
		buf = h.appendAttr(buf, a)
	}

	r.Attrs(func(a slog.Attr) bool {
		if h.isTraceKey(a.Key) {
			if tracePos >= 0 {
				traceID := h.traceValueFromAttr(a)
				if traceID != "" && traceID != "-" {
					buf = replaceBytes(buf, tracePos, 1, traceID)
				}
				tracePos = -1
			}
			return true
		}
		buf = h.appendAttr(buf, a)
		return true
	})

	buf = append(buf, '\n')

	h.mu.Lock()
	_, err := h.w.Write(buf)
	h.mu.Unlock()

	if cap(buf) > maxTextBufCap {
		tb.b = make([]byte, 0, 256)
	} else {
		tb.b = buf[:0]
	}
	textBufPool.Put(tb)
	return err
}

func (h *textHandler) handleEntry(t time.Time, level slog.Level, msg string, attrs []slog.Attr) error {
	tb := textBufPool.Get().(*textBuf)
	buf := tb.b[:0]

	if !t.IsZero() {
		buf = t.AppendFormat(buf, h.timeFormat)
	}

	msg = trimSpaceFast(msg)
	if msg == "" {
		msg = "-"
	}

	buf = append(buf, h.sep...)
	buf = append(buf, levelString(level)...)
	buf = append(buf, h.sep...)
	tracePos := len(buf)
	buf = append(buf, '-')
	buf = append(buf, h.sep...)
	buf = append(buf, msg...)

	for _, a := range h.attrs {
		if h.isTraceKey(a.Key) {
			if tracePos >= 0 {
				traceID := h.traceValueFromAttr(a)
				if traceID != "" && traceID != "-" {
					buf = replaceBytes(buf, tracePos, 1, traceID)
				}
				tracePos = -1
			}
			continue
		}
		buf = h.appendAttr(buf, a)
	}

	for _, a := range attrs {
		if h.isTraceKey(a.Key) {
			if tracePos >= 0 {
				traceID := h.traceValueFromAttr(a)
				if traceID != "" && traceID != "-" {
					buf = replaceBytes(buf, tracePos, 1, traceID)
				}
				tracePos = -1
			}
			continue
		}
		buf = h.appendAttr(buf, a)
	}

	buf = append(buf, '\n')

	h.mu.Lock()
	_, err := h.w.Write(buf)
	h.mu.Unlock()

	if cap(buf) > maxTextBufCap {
		tb.b = make([]byte, 0, 256)
	} else {
		tb.b = buf[:0]
	}
	textBufPool.Put(tb)
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.mu = &sync.Mutex{}
	next.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &next
}

func (h *textHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.mu = &sync.Mutex{}
	next.groups = append(append([]string(nil), h.groups...), name)
	return &next
}

func (h *textHandler) isTraceKey(key string) bool {
	if key == "" {
		return false
	}
	tk := h.traceKeys
	return key == tk.k0 || key == tk.k1 || key == tk.k2
}

func (h *textHandler) findTraceID(r slog.Record) string {
	for _, a := range h.attrs {
		if v := h.traceFromAttr(a); v != "" {
			return v
		}
	}

	var found string
	r.Attrs(func(a slog.Attr) bool {
		if found != "" {
			return false
		}
		found = h.traceFromAttr(a)
		return found == ""
	})
	return found
}

func (h *textHandler) traceFromAttr(a slog.Attr) string {
	if !h.isTraceKey(a.Key) {
		return ""
	}
	return h.traceValueFromAttr(a)
}

func (h *textHandler) traceValueFromAttr(a slog.Attr) string {
	v := a.Value
	if v.Kind() == slog.KindAny {
		v = v.Resolve()
	}
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindAny:
		if s, ok := v.Any().(string); ok {
			return s
		}
	}
	return formatTextValue(v)
}

func (h *textHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	if a.Key == "" {
		return buf
	}

	v := a.Value
	if v.Kind() == slog.KindAny {
		v = v.Resolve()
	}

	if v.Kind() == slog.KindGroup {
		groupPrefix := h.attrKey(a.Key)
		for _, sub := range v.Group() {
			if sub.Value.Kind() == slog.KindAny {
				sub.Value = sub.Value.Resolve()
			}
			if sub.Key == "" {
				continue
			}
			key := groupPrefix + "." + sub.Key
			if sub.Value.Kind() == slog.KindGroup {
				buf = h.appendAttr(buf, slog.Attr{Key: key, Value: sub.Value})
				continue
			}
			buf = append(buf, h.sep...)
			buf = append(buf, key...)
			buf = append(buf, '=')
			buf = appendTextValue(buf, sub.Value)
		}
		return buf
	}

	buf = append(buf, h.sep...)
	buf = append(buf, h.attrKey(a.Key)...)
	buf = append(buf, '=')
	buf = appendTextValue(buf, v)
	return buf
}

func (h *textHandler) attrKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	var b strings.Builder
	for i, g := range h.groups {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(g)
	}
	b.WriteByte('.')
	b.WriteString(key)
	return b.String()
}

func appendTextValue(dst []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return append(dst, v.String()...)
	case slog.KindInt64:
		return strconv.AppendInt(dst, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(dst, v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.AppendFloat(dst, v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		if v.Bool() {
			return append(dst, "true"...)
		}
		return append(dst, "false"...)
	case slog.KindDuration:
		return append(dst, v.Duration().String()...)
	case slog.KindTime:
		t := v.Time()
		if t.IsZero() {
			return dst
		}
		return t.AppendFormat(dst, time.RFC3339Nano)
	case slog.KindAny:
		a := v.Any()
		switch raw := a.(type) {
		case json.RawMessage:
			return append(dst, raw...)
		case error:
			return append(dst, raw.Error()...)
		}

		b, err := json.Marshal(a)
		if err == nil {
			return append(dst, b...)
		}
		return append(dst, fmt.Sprint(a)...)
	default:
		return append(dst, v.String()...)
	}
}

func formatTextValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		t := v.Time()
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339Nano)
	case slog.KindAny:
		a := v.Any()
		switch raw := a.(type) {
		case json.RawMessage:
			return string(raw)
		case error:
			return raw.Error()
		}

		b, err := json.Marshal(a)
		if err == nil {
			return string(b)
		}
		return fmt.Sprint(a)
	default:
		return v.String()
	}
}
