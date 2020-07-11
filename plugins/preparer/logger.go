package preparer

import (
	"log"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-hclog"
)

var _ hclog.Logger = &logrLogger{}

type logrLogger struct {
	logger logr.Logger
}

func (l *logrLogger) Trace(msg string, args ...interface{}) {
	l.logger.V(1).Info(msg, args...)
}

func (l *logrLogger) Debug(msg string, args ...interface{}) {
	l.logger.V(1).Info(msg, args...)
}

func (l *logrLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l *logrLogger) Warn(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l *logrLogger) Error(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l *logrLogger) IsTrace() bool {
	return l.logger.Enabled()
}

func (l *logrLogger) IsDebug() bool {
	return l.logger.Enabled()
}

func (l *logrLogger) IsInfo() bool {
	return l.logger.Enabled()
}

func (l *logrLogger) IsWarn() bool {
	return l.logger.Enabled()
}

func (l *logrLogger) IsError() bool {
	return l.logger.Enabled()
}

func (l *logrLogger) With(args ...interface{}) hclog.Logger {
	return &logrLogger{l.logger.WithValues(args...)}
}

func (l *logrLogger) Named(name string) hclog.Logger {
	return &logrLogger{l.logger.WithName(name)}
}

func (l *logrLogger) ResetNamed(_ string) hclog.Logger {
	return &logrLogger{l.logger}
}

func (l *logrLogger) SetLevel(_ hclog.Level) {
}

func (l *logrLogger) StandardLogger(_ *hclog.StandardLoggerOptions) *log.Logger {
	return nil
}
