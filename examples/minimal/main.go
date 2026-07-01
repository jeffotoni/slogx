package main

import (
	"fmt"

	"github.com/jeffotoni/log"
)

func main() {
	if err := log.New().
		Info().
		Str("component", "bootstrap").
		Int("attempt", 1).
		Msg("service started").
		Send(); err != nil {
		fmt.Println("log error:", err)
	}
}
