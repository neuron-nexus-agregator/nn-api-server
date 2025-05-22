package main

import (
	"agregator/api/internal/pkg/app"
	"log/slog"
)

func main() {
	logger := slog.Default
	app := app.New(logger())
	app.Run()
}
