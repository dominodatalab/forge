package util

import "github.com/go-logr/logr"

// logr.Logger writer interface adapter
// LogrWriter is an
type LogrWriter struct {
	Logger logr.Logger
}

// Write logs a message at the info level.
func (w *LogrWriter) Write(msg []byte) (int, error) {
	w.Logger.Info(string(msg))
	return len(msg), nil
}
