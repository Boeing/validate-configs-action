package main

import (
	"os"

	"github.com/Boeing/validate-configs-action/internal/config"
	"github.com/Boeing/validate-configs-action/internal/runner"
)

func main() {
	cfg := config.Load()
	os.Exit(runner.Run(&cfg))
}
