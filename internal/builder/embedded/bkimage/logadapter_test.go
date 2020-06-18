package bkimage

import (
	"testing"

	"github.com/go-logr/logr"
)

type fakeLogger struct{ msg string }

func (l *fakeLogger) Enabled() bool                         { return true }
func (l *fakeLogger) Info(msg string, kv ...interface{})    { l.msg = msg }
func (l *fakeLogger) Error(error, string, ...interface{})   {}
func (l *fakeLogger) V(int) logr.InfoLogger                 { return nil }
func (l *fakeLogger) WithValues(...interface{}) logr.Logger { return nil }
func (l *fakeLogger) WithName(string) logr.Logger           { return nil }

func TestLogrWriter_Write(t *testing.T) {
	l := &fakeLogger{}
	w := LogrWriter{l}
	msg := "hello, steve"
	size, err := w.Write([]byte(msg))

	if err != nil {
		t.Fatalf("no error was expected: %v", err)
	}

	if size != len(msg) {
		t.Fatalf("expected message size %d, got %d", len(msg), size)
	}

	if l.msg != msg {
		t.Fatalf("expected logged info message %q, got %q", msg, l.msg)
	}
}
