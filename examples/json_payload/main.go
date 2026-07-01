package main

import (
	"fmt"
	"os"

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
		JSON("payload", []byte(`{"event":"user.created","ok":true}`)).
		Msg("explicit JSON field").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Info().
		Any("payload", []byte(`{"event":"queue.send","size":128}`)).
		Msg("Any auto-detects JSON bytes").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Info().
		Any("payload", []byte("not-json")).
		Msg("invalid JSON bytes fall back to string").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
