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
		Level:       log.INFO,
		ServiceName: "api",
	})

	login(logger)
}

func login(logger *log.Logger) {
	if err := logger.Info().
		Caller().
		Component("auth").
		Action("login").
		Msg("caller field included automatically").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
