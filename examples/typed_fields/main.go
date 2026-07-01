package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jeffotoni/log"
)

func main() {
	logger := log.New(log.Config{
		Format:      log.FormatJSON,
		Writer:      os.Stdout,
		Level:       log.INFO,
		ServiceName: "api",
	})

	if err := logger.Info().
		Bool("success", true).
		Int("status", 200).
		Int64("bytes", 1234).
		Float64("latency_ms", 12.3).
		Number("retries", uint(2)).
		Duration("elapsed", 120*time.Millisecond).
		Time("finished_at", time.Now(), log.LayoutISO8601Nano).
		Msg("typed fields example").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
