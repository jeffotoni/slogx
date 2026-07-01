package main

import (
	"errors"
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

	if err := logger.Error().
		Err(errors.New("database unavailable")).
		Msg("request failed").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Warn().
		Err("cause", errors.New("dependency timeout")).
		Msg("dependency degraded").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
