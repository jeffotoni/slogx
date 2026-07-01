
// Example:
// curl -i -XPOST -H "Content-Type: application/json" localhost:8080/v1/user -d '{"name":"jeff","year":2026}'
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jeffotoni/slogx"
)

const traceHeader = "X-Trace-ID"

type createUserRequest struct {
	Name string `json:"name"`
	Year int    `json:"year"`
}

type createUserResponse struct {
	OK      bool   `json:"ok"`
	TraceID string `json:"traceId"`
}

var log = slogx.New(slogx.Config{
	Format:      slogx.FormatJSON,
	Writer:      os.Stdout,
	TimeFormat:  slogx.LayoutISO8601Nano,
	Level:       slogx.DEBUG,
	ServiceName: "api-user",
	TraceIDKey:  traceHeader,
})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user", handleCreateUser)

	addr := ":8080"
	if err := log.Info().
		Action("startup").
		Str("addr", addr).
		Msg("http server starting").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "log error:", err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = log.Error().
			Action("startup").
			Err("error", err).
			Msg("http server stopped unexpectedly").
			Send()
		os.Exit(1)
	}
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get(traceHeader)
	if traceID == "" {
		traceID = newTraceID()
	}

	ctx, cancel := slogx.NewCtx(r.Context()).
		TraceKey(traceHeader).
		TraceID(traceID).
		Set("X-User-ID", "user3039").
		Set("X-Span-ID", "span39393").
		Timeout(10 * time.Second).
		Build()
	defer cancel()

	ctx = slogx.WithCtx(ctx).
		Any("attempt", 1).
		Bool("cached", false).
		Context()

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if sendErr := log.Error().
			Ctx(ctx).
			Component("http").
			Action("decode_body").
			Err("error", err).
			Msg("invalid request body").
			Send(); sendErr != nil {
			fmt.Fprintln(os.Stderr, "log error:", sendErr)
		}
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := log.Debug().
		Ctx(ctx).
		Component("http").
		Action("validate").
		Str("name", req.Name).
		Int("year", req.Year).
		Msg("request decoded").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "log error:", err)
	}

	if err := saveSomeWhere(ctx, req); err != nil {
		if sendErr := log.Error().
			Ctx(ctx).
			Component("handler").
			Action("save").
			Err("error", err).
			Msg("failed to persist user").
			Send(); sendErr != nil {
			fmt.Fprintln(os.Stderr, "log error:", sendErr)
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := createUserResponse{
		OK:      true,
		TraceID: traceID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(traceHeader, traceID)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)

	if err := log.Info().
		Ctx(ctx).
		Component("handler").
		Action("response").
		Int("status", http.StatusOK).
		Msg("request completed").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "log error:", err)
	}
}

func saveSomeWhere(ctx context.Context, req createUserRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err := log.Debug().
		Ctx(ctx).
		Component("storage").
		Action("marshal").
		Int("bytes", len(payload)).
		Msg("payload marshaled").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "log error:", err)
	}

	return sendQueue(ctx, payload)
}

func sendQueue(ctx context.Context, payload []byte) error {
	time.Sleep(50 * time.Millisecond)

	if err := log.Debug().
		Ctx(ctx).
		Component("queue").
		Action("send").
		Int("bytes", len(payload)).
		Msg("queue send success").
		Send(); err != nil {
		fmt.Fprintln(os.Stderr, "log error:", err)
	}

	return nil
}

func newTraceID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconvFallbackTraceID()
	}
	return hex.EncodeToString(b[:])
}

func strconvFallbackTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
