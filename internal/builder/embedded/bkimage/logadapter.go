package bkimage

import "github.com/go-logr/logr"

// LogrWriter is an interface adapter for logr.Logger and io.Writer
type LogrWriter struct {
	Logger logr.Logger
}

// Write logs a message at the info level.
func (w *LogrWriter) Write(msg []byte) (int, error) {
	w.Logger.Info(string(msg))
	return len(msg), nil
}
