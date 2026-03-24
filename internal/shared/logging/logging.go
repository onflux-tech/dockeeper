package logging

import (
	"log/slog"
	"os"
)

// ConfigureDefault installs a text slog logger that writes regular logs to stdout.
func ConfigureDefault() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))
}
