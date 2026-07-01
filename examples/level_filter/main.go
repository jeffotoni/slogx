package main

import (
	"fmt"
	"os"

	"github.com/jeffotoni/log"
)

func main() {
	logger := log.New(log.Config{
		Format:      log.FormatText,
		Writer:      os.Stdout,
		Level:       log.WARN,
		ServiceName: "api",
	})

	fmt.Println("only WARN and ERROR entries are emitted below")

	if err := logger.Debug().
		Msg("debug is filtered out").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Info().
		Msg("info is filtered out").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Warn().
		Msg("warn is emitted").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}

	if err := logger.Error().
		Msg("error is emitted").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
