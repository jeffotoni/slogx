package slogx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const maxJSONBufCap = 64 * 1024

type jsonBuf struct {
	b []byte
}

var jsonBufPool = sync.Pool{
	New: func() any {
		return &jsonBuf{b: make([]byte, 0, 256)}
	},
}

type jsonHandler struct {
	w          io.Writer
	minLevel   slog.Level
	timeFormat string

	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func newJSONHandler(cfg Config, _ *slog.HandlerOptions) slog.Handler {
	return &jsonHandler{
		w:          cfg.Writer,
		minLevel:   toSlogLevel(cfg.Level),
		timeFormat: cfg.TimeFormat,
		mu:         &sync.Mutex{},
	}
}

func (h *jsonHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

func (h *jsonHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(nil, r.Level) {
		return nil
	}

	jb := jsonBufPool.Get().(*jsonBuf)
	buf := jb.b[:0]

	buf = append(buf, '{')
	first := true

	// time
	buf, first = h.appendTimeField(buf, first, slog.TimeKey, r.Time)
	// level
	buf, first = h.appendStringField(buf, first, slog.LevelKey, levelString(r.Level))
	// msg
	buf, first = h.appendStringField(buf, first, slog.MessageKey, r.Message)

	for _, a := range h.attrs {
		buf, first = h.appendAttr(buf, first, a)
	}

	r.Attrs(func(a slog.Attr) bool {
		buf, first = h.appendAttr(buf, first, a)
		return true
	})

	buf = append(buf, '}', '\n')

	h.mu.Lock()
	_, err := h.w.Write(buf)
	h.mu.Unlock()

	if cap(buf) > maxJSONBufCap {
		jb.b = make([]byte, 0, 256)
	} else {
		jb.b = buf[:0]
	}
	jsonBufPool.Put(jb)
	return err
}

func (h *jsonHandler) handleEntry(t time.Time, level slog.Level, msg string, attrs []slog.Attr) error {
	jb := jsonBufPool.Get().(*jsonBuf)
	buf := jb.b[:0]

	buf = append(buf, '{')
	first := true

	// time
	buf, first = h.appendTimeField(buf, first, slog.TimeKey, t)
	// level
	buf, first = h.appendStringField(buf, first, slog.LevelKey, levelString(level))
	// msg
	buf, first = h.appendStringField(buf, first, slog.MessageKey, msg)

	for _, a := range h.attrs {
		buf, first = h.appendAttr(buf, first, a)
	}

	for _, a := range attrs {
		buf, first = h.appendAttr(buf, first, a)
	}

	buf = append(buf, '}', '\n')

	h.mu.Lock()
	_, err := h.w.Write(buf)
	h.mu.Unlock()

	if cap(buf) > maxJSONBufCap {
		jb.b = make([]byte, 0, 256)
	} else {
		jb.b = buf[:0]
	}
	jsonBufPool.Put(jb)
	return err
}

func (h *jsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.mu = &sync.Mutex{}
	next.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &next
}

func (h *jsonHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.mu = &sync.Mutex{}
	next.groups = append(append([]string(nil), h.groups...), name)
	return &next
}

func (h *jsonHandler) appendTimeField(dst []byte, first bool, key string, t time.Time) ([]byte, bool) {
	if t.IsZero() {
		return dst, first
	}
	if !first {
		dst = append(dst, ',')
	}
	dst = appendJSONString(dst, key)
	dst = append(dst, ':', '"')
	dst = t.AppendFormat(dst, h.timeFormat)
	dst = append(dst, '"')
	return dst, false
}

func (h *jsonHandler) appendStringField(dst []byte, first bool, key, val string) ([]byte, bool) {
	if key == "" {
		return dst, first
	}
	if !first {
		dst = append(dst, ',')
	}
	dst = appendJSONString(dst, key)
	dst = append(dst, ':')
	dst = appendJSONString(dst, val)
	return dst, false
}

func (h *jsonHandler) appendAttr(dst []byte, first bool, a slog.Attr) ([]byte, bool) {
	if a.Key == "" {
		return dst, first
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
			dst, first = h.appendAttr(dst, first, slog.Attr{Key: key, Value: sub.Value})
		}
		return dst, first
	}

	if !first {
		dst = append(dst, ',')
	}
	dst = appendJSONString(dst, h.attrKey(a.Key))
	dst = append(dst, ':')
	dst = appendJSONValue(dst, v)
	return dst, false
}

func (h *jsonHandler) attrKey(key string) string {
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

func appendJSONValue(dst []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return appendJSONString(dst, v.String())
	case slog.KindInt64:
		return strconv.AppendInt(dst, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(dst, v.Uint64(), 10)
	case slog.KindFloat64:
		return appendJSONFloat64(dst, v.Float64())
	case slog.KindBool:
		return strconv.AppendBool(dst, v.Bool())
	case slog.KindDuration:
		return strconv.AppendInt(dst, int64(v.Duration()), 10)
	case slog.KindTime:
		t := v.Time()
		dst = append(dst, '"')
		dst = t.AppendFormat(dst, time.RFC3339Nano)
		dst = append(dst, '"')
		return dst
	case slog.KindAny:
		a := v.Any()
		switch raw := a.(type) {
		case json.RawMessage:
			return append(dst, raw...)
		case error:
			return appendJSONString(dst, raw.Error())
		}

		// Match slog.JSONHandler (EscapeHTML=false) when possible.
		var bb bytes.Buffer
		enc := json.NewEncoder(&bb)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(a); err != nil {
			return appendJSONString(dst, fmt.Sprint(a))
		}
		bs := bb.Bytes()
		// Remove trailing newline that Encoder adds.
		if len(bs) > 0 && bs[len(bs)-1] == '\n' {
			bs = bs[:len(bs)-1]
		}
		return append(dst, bs...)
	default:
		return appendJSONString(dst, v.String())
	}
}

const hexChars = "0123456789abcdef"

func appendJSONFloat64(dst []byte, f float64) []byte {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		// Match slog's behavior (encoding/json error -> "!ERROR:...").
		// Example: "json: unsupported value: NaN".
		err := errors.New("json: unsupported value: " + strconv.FormatFloat(f, 'g', -1, 64))
		return appendJSONString(dst, "!ERROR:"+err.Error())
	}

	// Match encoding/json float formatting.
	abs := math.Abs(f)
	fmtByte := byte('f')
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		fmtByte = 'e'
	}
	dst = strconv.AppendFloat(dst, f, fmtByte, -1, 64)
	if fmtByte == 'e' {
		// clean up e-09 to e-9
		n := len(dst)
		if n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst
}

func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	dst = appendEscapedJSONString(dst, s)
	dst = append(dst, '"')
	return dst
}

func appendEscapedJSONString(dst []byte, s string) []byte {
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			// Fast path: no escaping.
			if b >= 0x20 && b != '\\' && b != '"' {
				i++
				continue
			}
			if start < i {
				dst = append(dst, s[start:i]...)
			}
			dst = append(dst, '\\')
			switch b {
			case '\\', '"':
				dst = append(dst, b)
			case '\n':
				dst = append(dst, 'n')
			case '\r':
				dst = append(dst, 'r')
			case '\t':
				dst = append(dst, 't')
			default:
				dst = append(dst, 'u', '0', '0', hexChars[b>>4], hexChars[b&0xF])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if start < i {
				dst = append(dst, s[start:i]...)
			}
			dst = append(dst, `\ufffd`...)
			i += size
			start = i
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			if start < i {
				dst = append(dst, s[start:i]...)
			}
			dst = append(dst, `\u202`...)
			dst = append(dst, hexChars[r&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		dst = append(dst, s[start:]...)
	}
	return dst
}
