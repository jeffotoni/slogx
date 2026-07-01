package slogx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffotoni/slogx"
)

type failWriter struct {
	err error
}

func (w failWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestNew_DefaultConfig(t *testing.T) {
	logger := slogx.New()
	if logger == nil {
		t.Fatal("expected logger instance, got nil")
	}

	v := reflect.ValueOf(logger).Elem()
	cfg := v.FieldByName("cfg")

	if cfg.FieldByName("Writer").IsNil() {
		t.Error("expected default Writer (os.Stdout), got nil")
	}
	if cfg.FieldByName("TimeFormat").String() != slogx.LayoutDefault {
		t.Errorf("expected default TimeFormat=%s, got %s", slogx.LayoutDefault, cfg.FieldByName("TimeFormat").String())
	}
	if slogx.Level(cfg.FieldByName("Level").String()) != slogx.INFO {
		t.Errorf("expected default Level=INFO, got %s", cfg.FieldByName("Level").String())
	}
	if slogx.Format(cfg.FieldByName("Format").String()) != slogx.FormatJSON {
		t.Errorf("expected default Format=json, got %s", cfg.FieldByName("Format").String())
	}
}

func TestJSON_OutputIncludesField(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Debug().Str("event", "test").Msg("ok").Send()

	if !strings.Contains(buf.String(), `"event":"test"`) {
		t.Fatalf("expected field in JSON output, got: %s", buf.String())
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.WARN,
	})

	logger.Debug().Str("k", "v").Msg("should not log").Send()
	if buf.Len() != 0 {
		t.Fatalf("expected no output for DEBUG below WARN, got: %s", buf.String())
	}

	logger.Error().Str("k", "v").Msg("should log").Send()
	if buf.Len() == 0 {
		t.Fatalf("expected output for ERROR at WARN min level, got empty")
	}
}

func TestTimeFormatOverride(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format:     slogx.FormatJSON,
		Writer:     &buf,
		Level:      slogx.DEBUG,
		TimeFormat: slogx.LayoutDateTime,
	})

	logger.Info().Msg("t").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}

	timeStr, _ := m["time"].(string)
	if timeStr == "" {
		t.Fatalf("expected time field in output, got: %v", m)
	}
	if _, err := time.Parse(slogx.LayoutDateTime, timeStr); err != nil {
		t.Fatalf("expected time to parse with layout %q, got %q: %v", slogx.LayoutDateTime, timeStr, err)
	}
}

func TestTraceLevelRendersAsTRACE(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.TRACE,
	})

	logger.Trace().Msg("t").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}

	if got, _ := m["level"].(string); got != "TRACE" {
		t.Fatalf("expected level=TRACE, got %q", got)
	}
}

func TestJSONField_EmbedsObject(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().JSON("payload", []byte(`{"a":1}`)).Msg("x").Send()
	out := buf.String()
	if !strings.Contains(out, `"payload":{"a":1}`) {
		t.Fatalf("expected embedded json object in output, got: %s", out)
	}
}

func TestJSONField_InvalidJSONFallsBackToString(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().JSON("payload", []byte("{not json")).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected valid json output, got unmarshal error: %v (raw=%q)", err, buf.String())
	}

	if _, ok := m["payload"].(string); !ok {
		t.Fatalf("expected payload to fallback to string, got: %#v", m["payload"])
	}
}

func TestAny_MapIsSerialized(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().Any("data", map[string]int{"a": 1}).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	data, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data as object, got %#v", m["data"])
	}
	if data["a"] != float64(1) {
		t.Fatalf("expected data.a=1, got %#v", data["a"])
	}
}

func TestAny_BytesAutoDetectJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().Any("payload", []byte(`{"a":1}`)).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if _, ok := m["payload"].(map[string]any); !ok {
		t.Fatalf("expected payload as object, got %#v", m["payload"])
	}
}

func TestAny_BytesFallbackToString(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().Any("payload", []byte("hello")).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if got, ok := m["payload"].(string); !ok || got != "hello" {
		t.Fatalf("expected payload=hello, got %#v", m["payload"])
	}
}

func TestCtx_ImportsFieldsIntoEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	ctx, cancel := slogx.NewCtx().Set("X-User-ID", "user42").Build()
	defer cancel()
	ctx = slogx.WithCtx(ctx).Any("attempt", 3).Context()

	logger.Info().Ctx(ctx).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}

	if got, _ := m["X-User-ID"].(string); got != "user42" {
		t.Fatalf("expected X-User-ID=user42, got %#v", m["X-User-ID"])
	}
	if got, _ := m["attempt"].(float64); got != 3 {
		t.Fatalf("expected attempt=3, got %#v", m["attempt"])
	}
}

func TestFormats_TextAndSlog(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slogx.New(slogx.Config{
			Format:    slogx.FormatText,
			Writer:    &buf,
			Level:     slogx.DEBUG,
			Separator: " | ",
		})

		logger.Info().TraceID("abc123").Str("k", "v").Msg("m").Send()
		out := buf.String()
		if !strings.Contains(out, " | INFO | abc123 | m") {
			t.Fatalf("unexpected text header output: %q", out)
		}
		if strings.Contains(out, "traceId=") {
			t.Fatalf("expected traceId to not be duplicated in fields: %q", out)
		}
		if !strings.Contains(out, "k=v") {
			t.Fatalf("unexpected text output: %q", out)
		}
	})

	t.Run("slog", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slogx.New(slogx.Config{
			Format: slogx.FormatSlog,
			Writer: &buf,
			Level:  slogx.DEBUG,
		})

		logger.Info().TraceID("abc123").Str("k", "v").Msg("m").Send()
		out := buf.String()
		if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "k=v") || !strings.Contains(out, "msg=m") || !strings.Contains(out, "traceId=abc123") {
			t.Fatalf("unexpected slog output: %q", out)
		}
	})
}

func TestEntry_UsesProvidedContext(t *testing.T) {
	var got any
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	ctx = slogx.WithCtx(ctx).Any("x", "y").Context()

	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().Ctx(ctx).Msg("x").Send()

	var m map[string]any
	_ = json.Unmarshal(buf.Bytes(), &m)
	got = m["x"]
	if got != "y" {
		t.Fatalf("expected context field to be present, got %#v", got)
	}
}

func TestEntry_ErrDefaultKey(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Error().Err(errors.New("boom")).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if got, _ := m["error"].(string); got != "boom" {
		t.Fatalf("expected error=boom, got %#v", m["error"])
	}
}

func TestEntry_ErrCustomKey(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Error().Err("db_err", errors.New("boom")).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if got, _ := m["db_err"].(string); got != "boom" {
		t.Fatalf("expected db_err=boom, got %#v", m["db_err"])
	}
}

func TestEntry_Action(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().Action("login").Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if got, _ := m["action"].(string); got != "login" {
		t.Fatalf("expected action=login, got %#v", m["action"])
	}
}

func TestEntry_Time(t *testing.T) {
	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	tm := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	logger.Info().Time("at", tm).Msg("x").Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}
	if got, _ := m["at"].(string); got != tm.Format(time.RFC3339) {
		t.Fatalf("expected at=%q, got %#v", tm.Format(time.RFC3339), m["at"])
	}
}

func TestEntry_Number(t *testing.T) {
	type myInt int

	var buf bytes.Buffer
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: &buf,
		Level:  slogx.DEBUG,
	})

	logger.Info().
		Number("status", 200).
		Number("bytes", int64(1234)).
		Number("u", uint32(9)).
		Number("latency_ms", float32(12.3)).
		Number("bad", "x").
		Number("bool", true).
		Number("custom", myInt(7)).
		Number("", 999).
		Msg("x").
		Send()

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal json: %v (raw=%q)", err, buf.String())
	}

	if got, ok := m["status"].(float64); !ok || got != 200 {
		t.Fatalf("expected status=200, got %#v", m["status"])
	}
	if got, ok := m["bytes"].(float64); !ok || got != 1234 {
		t.Fatalf("expected bytes=1234, got %#v", m["bytes"])
	}
	if got, ok := m["u"].(float64); !ok || got != 9 {
		t.Fatalf("expected u=9, got %#v", m["u"])
	}
	if got, ok := m["latency_ms"].(float64); !ok || math.Abs(got-12.3) > 1e-6 {
		t.Fatalf("expected latency_ms≈12.3, got %#v", m["latency_ms"])
	}

	if _, ok := m["bad"]; ok {
		t.Fatalf("expected bad to be ignored, got %#v", m["bad"])
	}
	if _, ok := m["bool"]; ok {
		t.Fatalf("expected bool to be ignored, got %#v", m["bool"])
	}
	if _, ok := m["custom"]; ok {
		t.Fatalf("expected custom to be ignored, got %#v", m["custom"])
	}
	if _, ok := m[""]; ok {
		t.Fatalf("expected empty key to be ignored, got %#v", m[""])
	}
}

func TestSend_ReturnsWriterError(t *testing.T) {
	expectedErr := errors.New("write failed")
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: failWriter{err: expectedErr},
		Level:  slogx.DEBUG,
	})

	err := logger.Info().Msg("x").Send()
	if err == nil {
		t.Fatalf("expected send error, got nil")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf("expected error to contain %q, got %v", expectedErr.Error(), err)
	}
}

func TestSend_DisabledLevelSkipsWriterAndReturnsNil(t *testing.T) {
	logger := slogx.New(slogx.Config{
		Format: slogx.FormatJSON,
		Writer: failWriter{err: errors.New("write failed")},
		Level:  slogx.ERROR,
	})

	err := logger.Info().Msg("x").Send()
	if err != nil {
		t.Fatalf("expected nil error when level is disabled, got %v", err)
	}
}
