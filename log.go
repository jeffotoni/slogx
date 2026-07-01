package log

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
	FormatSlog Format = "slog"
)

type Level string

const (
	TRACE Level = "TRACE"
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

const (
	LayoutDefault     = time.RFC3339      // 2006-01-02T15:04:05Z07:00
	LayoutCompact     = "20060102T150405" // 20250101T150405
	LayoutDateTime    = "2006-01-02 15:04:05"
	LayoutDateOnly    = "2006-01-02"
	LayoutTimeOnly    = "15:04:05"
	LayoutISO8601Nano = time.RFC3339Nano
)

const DefaultTraceIDKey = "traceId"

type Config struct {
	Format     Format
	Writer     io.Writer
	TimeFormat string
	Level      Level
	Separator  string

	ServiceName string
	TraceIDKey  string
}

type Logger struct {
	cfg  Config
	slog *slog.Logger
}

type Entry struct {
	level     Level
	msg       string
	attrs     []slog.Attr
	logger    *Logger
	addCaller bool
	ctx       context.Context
}

var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{attrs: make([]slog.Attr, 0, 8)}
	},
}

func getEntry() *Entry {
	e := entryPool.Get().(*Entry)
	for i := range e.attrs {
		e.attrs[i] = slog.Attr{}
	}
	e.attrs = e.attrs[:0]
	e.level = ""
	e.msg = ""
	e.logger = nil
	e.addCaller = false
	e.ctx = nil
	return e
}

func putEntry(e *Entry) {
	e.ctx = nil
	entryPool.Put(e)
}

func normalizeConfig(cfg Config) Config {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = LayoutDefault
	}
	if cfg.Level == "" {
		cfg.Level = INFO
	}
	if cfg.Format == "" {
		cfg.Format = FormatJSON
	}
	if cfg.Separator == "" {
		if cfg.Format == FormatText {
			cfg.Separator = " | "
		} else {
			cfg.Separator = " "
		}
	}
	if cfg.TraceIDKey == "" {
		cfg.TraceIDKey = DefaultTraceIDKey
	}
	return cfg
}

func toSlogLevel(min Level) slog.Level {
	switch min {
	case TRACE:
		return slog.LevelDebug - 4
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l Level) slogLevel() slog.Level {
	switch l {
	case TRACE:
		return slog.LevelDebug - 4
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func replaceAttr(cfg Config) func(_ []string, a slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.TimeKey:
			t := a.Value.Time()
			if t.IsZero() {
				return slog.String(slog.TimeKey, "")
			}
			return slog.String(slog.TimeKey, t.Format(cfg.TimeFormat))
		case slog.LevelKey:
			switch a.Value.Kind() {
			case slog.KindAny:
				if lv, ok := a.Value.Any().(slog.Level); ok {
					return slog.String(slog.LevelKey, levelString(lv))
				}
			case slog.KindInt64:
				return slog.String(slog.LevelKey, levelString(slog.Level(a.Value.Int64())))
			}
			return slog.String(slog.LevelKey, a.Value.String())
		default:
			return a
		}
	}
}

func levelString(lv slog.Level) string {
	switch lv {
	case slog.LevelDebug - 4:
		return string(TRACE)
	case slog.LevelDebug:
		return string(DEBUG)
	case slog.LevelInfo:
		return string(INFO)
	case slog.LevelWarn:
		return string(WARN)
	case slog.LevelError:
		return string(ERROR)
	default:
		return lv.String()
	}
}

func newSlogLogger(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       toSlogLevel(cfg.Level),
		ReplaceAttr: replaceAttr(cfg),
	}

	var h slog.Handler
	switch cfg.Format {
	case FormatSlog:
		h = slog.NewTextHandler(cfg.Writer, opts)
	case FormatText:
		h = newTextHandler(cfg, opts)
	default:
		h = newJSONHandler(cfg, opts)
	}

	return slog.New(h)
}

func Set(cfg Config) *Logger {
	cfg = normalizeConfig(cfg)
	l := &Logger{
		cfg:  cfg,
		slog: newSlogLogger(cfg),
	}
	if cfg.ServiceName != "" {
		l.slog = l.slog.With("service", cfg.ServiceName)
	}
	return l
}

func New(cfgs ...Config) *Logger {
	var cfg Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return Set(cfg)
}

func (l *Logger) Trace() *Entry {
	e := getEntry()
	e.level = TRACE
	e.logger = l
	return e
}

func (l *Logger) Debug() *Entry {
	e := getEntry()
	e.level = DEBUG
	e.logger = l
	return e
}

func (l *Logger) Info() *Entry {
	e := getEntry()
	e.level = INFO
	e.logger = l
	return e
}

func (l *Logger) Warn() *Entry {
	e := getEntry()
	e.level = WARN
	e.logger = l
	return e
}

func (l *Logger) Error() *Entry {
	e := getEntry()
	e.level = ERROR
	e.logger = l
	return e
}

func (l *Logger) Service(name string) *Logger {
	if l == nil {
		return l
	}
	if name == "" || name == l.cfg.ServiceName {
		return l
	}
	cfg := l.cfg
	cfg.ServiceName = name
	return Set(cfg)
}

func (e *Entry) Caller() *Entry {
	e.addCaller = true
	return e
}

func (e *Entry) TraceID(id string) *Entry {
	if e == nil || e.logger == nil {
		return e
	}
	key := e.logger.cfg.TraceIDKey
	if key == "" {
		key = DefaultTraceIDKey
	}
	return e.Str(key, id)
}

func (e *Entry) Ctx(ctx context.Context) *Entry {
	if e == nil {
		return e
	}
	e.ctx = ctx
	if ctx == nil {
		return e
	}

	rawAny := ctx.Value(internalAnyFieldsKey)
	typed, _ := rawAny.(map[string]any)

	rawKeys := ctx.Value(internalKeysKey)
	keyList, ok := rawKeys.([]string)
	if ok && len(keyList) > 0 {
		for _, k := range keyList {
			if k == "" {
				continue
			}
			if typed != nil {
				if _, exists := typed[k]; exists {
					continue
				}
			}
			if s, ok := ctx.Value(getCtxKey(k)).(string); ok {
				e.Str(k, s)
			}
		}
	}

	for k, v := range typed {
		e.appendCtxAny(k, v)
	}
	return e
}

func (e *Entry) appendCtxAny(k string, v any) {
	if e == nil || k == "" {
		return
	}

	switch val := v.(type) {
	case nil:
		e.attrs = append(e.attrs, slog.Any(k, nil))
	case string:
		e.attrs = append(e.attrs, slog.String(k, val))
	case bool:
		e.attrs = append(e.attrs, slog.Bool(k, val))
	case time.Time:
		e.attrs = append(e.attrs, slog.Time(k, val))
	case time.Duration:
		e.attrs = append(e.attrs, slog.String(k, val.String()))
	case error:
		e.attrs = append(e.attrs, slog.String(k, val.Error()))
	case json.RawMessage:
		e.JSON(k, []byte(val))
	case []byte:
		e.Any(k, val)
	default:
		before := len(e.attrs)
		e.Number(k, val)
		if len(e.attrs) != before {
			return
		}
		e.attrs = append(e.attrs, slog.Any(k, val))
	}
}

func (e *Entry) Service(name string) *Entry {
	if name != "" {
		e.Str("service", name)
	}
	return e
}

func (e *Entry) Component(name string) *Entry {
	if name != "" {
		e.Str("component", name)
	}
	return e
}

func (e *Entry) Action(name string) *Entry {
	if name != "" {
		e.Str("action", name)
	}
	return e
}

func (e *Entry) Str(k, v string) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.String(k, v))
	return e
}

func (e *Entry) JSON(k string, data []byte) *Entry {
	if e == nil || k == "" {
		return e
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return e.Str(k, "")
	}

	if json.Valid(trimmed) {
		raw := make([]byte, len(trimmed))
		copy(raw, trimmed)
		e.attrs = append(e.attrs, slog.Any(k, json.RawMessage(raw)))
		return e
	}

	// Safe fallback: never emit invalid JSON.
	if utf8.Valid(trimmed) {
		return e.Str(k, string(trimmed))
	}
	return e.Str(k, base64.StdEncoding.EncodeToString(trimmed))
}

func (e *Entry) Any(k string, v any) *Entry {
	if e == nil || k == "" {
		return e
	}
	switch val := v.(type) {
	case nil:
		e.attrs = append(e.attrs, slog.Any(k, nil))
	case error:
		e.attrs = append(e.attrs, slog.String(k, val.Error()))
	case time.Duration:
		e.attrs = append(e.attrs, slog.String(k, val.String()))
	case json.RawMessage:
		return e.JSON(k, []byte(val))
	case []byte:
		trimmed := bytes.TrimSpace(val)
		if len(trimmed) == 0 {
			e.attrs = append(e.attrs, slog.String(k, ""))
			return e
		}
		if json.Valid(trimmed) {
			raw := make([]byte, len(trimmed))
			copy(raw, trimmed)
			e.attrs = append(e.attrs, slog.Any(k, json.RawMessage(raw)))
			return e
		}
		if utf8.Valid(trimmed) {
			e.attrs = append(e.attrs, slog.String(k, string(trimmed)))
			return e
		}
		e.attrs = append(e.attrs, slog.String(k, base64.StdEncoding.EncodeToString(trimmed)))
	default:
		e.attrs = append(e.attrs, slog.Any(k, v))
	}
	return e
}

func (e *Entry) Int(k string, v int) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.Int(k, v))
	return e
}

func (e *Entry) Int64(k string, v int64) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.Int64(k, v))
	return e
}

func (e *Entry) Float64(k string, v float64) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.Float64(k, v))
	return e
}

// Number is a convenience helper for logging native numeric values without choosing a typed method.
// Accepted types: int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64.
// Unsupported types are ignored silently.
//
// Example:
//
//	log.Info().Number("status", 200).Number("bytes", int64(1234)).Number("latency_ms", 12.3).Send()
func (e *Entry) Number(key string, value any) *Entry {
	if e == nil || key == "" {
		return e
	}

	switch v := value.(type) {
	case int:
		e.attrs = append(e.attrs, slog.Int64(key, int64(v)))
	case int8:
		e.attrs = append(e.attrs, slog.Int64(key, int64(v)))
	case int16:
		e.attrs = append(e.attrs, slog.Int64(key, int64(v)))
	case int32:
		e.attrs = append(e.attrs, slog.Int64(key, int64(v)))
	case int64:
		e.attrs = append(e.attrs, slog.Int64(key, v))
	case uint:
		e.attrs = append(e.attrs, slog.Uint64(key, uint64(v)))
	case uint8:
		e.attrs = append(e.attrs, slog.Uint64(key, uint64(v)))
	case uint16:
		e.attrs = append(e.attrs, slog.Uint64(key, uint64(v)))
	case uint32:
		e.attrs = append(e.attrs, slog.Uint64(key, uint64(v)))
	case uint64:
		e.attrs = append(e.attrs, slog.Uint64(key, v))
	case float32:
		e.attrs = append(e.attrs, slog.Float64(key, float64(v)))
	case float64:
		e.attrs = append(e.attrs, slog.Float64(key, v))
	}

	return e
}

func (e *Entry) Duration(k string, v time.Duration) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.String(k, v.String()))
	return e
}

func (e *Entry) Time(k string, v time.Time, layout ...string) *Entry {
	if e == nil || k == "" {
		return e
	}
	format := time.RFC3339
	if len(layout) > 0 && layout[0] != "" {
		format = layout[0]
	}
	e.attrs = append(e.attrs, slog.String(k, v.Format(format)))
	return e
}

func (e *Entry) Err(keyOrErr any, errs ...error) *Entry {
	if e == nil {
		return e
	}

	const defaultKey = "error"

	switch v := keyOrErr.(type) {
	case string:
		if v == "" {
			return e
		}
		for _, err := range errs {
			if err == nil {
				continue
			}
			return e.Any(v, err)
		}
		return e
	case error:
		if v != nil {
			return e.Any(defaultKey, v)
		}
		for _, err := range errs {
			if err == nil {
				continue
			}
			return e.Any(defaultKey, err)
		}
		return e
	default:
		for _, err := range errs {
			if err == nil {
				continue
			}
			return e.Any(defaultKey, err)
		}
		return e
	}
}

func (e *Entry) Bool(k string, v bool) *Entry {
	if e == nil || k == "" {
		return e
	}
	e.attrs = append(e.attrs, slog.Bool(k, v))
	return e
}

func (e *Entry) Msg(m string) *Entry {
	e.msg = m
	return e
}

func (e *Entry) Level() *Entry { return e }

func (e *Entry) Send() error {
	if e == nil || e.logger == nil || e.logger.slog == nil {
		return nil
	}
	defer putEntry(e)

	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	h := e.logger.slog.Handler()
	if h == nil {
		return nil
	}

	lv := e.level.slogLevel()
	if !h.Enabled(ctx, lv) {
		return nil
	}

	if e.addCaller {
		if _, file, line, ok := runtime.Caller(2); ok {
			e.attrs = append(e.attrs, slog.String("caller", file+":"+strconv.Itoa(line)))
		}
	}

	now := time.Now()
	var err error
	switch hh := h.(type) {
	case *textHandler:
		err = hh.handleEntry(now, lv, e.msg, e.attrs)
	case *jsonHandler:
		err = hh.handleEntry(now, lv, e.msg, e.attrs)
	default:
		r := slog.NewRecord(now, lv, e.msg, 0)
		r.AddAttrs(e.attrs...)
		err = h.Handle(ctx, r)
	}
	return err
}
