// Package nrzap supports https://github.com/uber-go/zap
//
// Wrap your zap Logger using nrzap.Transform to send agent log messages to zap.
package nrzap

import (
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
)

func init() { internal.TrackUsage("integration", "logging", "zap") }

type shim struct{ logger *zap.Logger }

func transformAttributes(atts map[string]interface{}) []zap.Field {
	fs := make([]zap.Field, 0, len(atts))
	for key, val := range atts {
		fs = append(fs, zap.Any(key, val))
	}
	return fs
}

func (s *shim) Error(msg string, c map[string]interface{}) {
	s.logger.Error(msg, transformAttributes(c)...)
}
func (s *shim) Warn(msg string, c map[string]interface{}) {
	s.logger.Warn(msg, transformAttributes(c)...)
}
func (s *shim) Info(msg string, c map[string]interface{}) {
	s.logger.Info(msg, transformAttributes(c)...)
}
func (s *shim) Debug(msg string, c map[string]interface{}) {
	s.logger.Debug(msg, transformAttributes(c)...)
}
func (s *shim) DebugEnabled() bool {
	ce := s.logger.Check(zap.DebugLevel, "debugging")
	return ce != nil
}

// Transform turns a *zap.Logger into a newrelic.Logger.
func Transform(l *zap.Logger) newrelic.Logger { return &shim{logger: l} }

// ConfigLogger configures the newrelic.Application to send log messsages to the
// provided zap logger.
func ConfigLogger(l *zap.Logger) newrelic.ConfigOption {
	return newrelic.ConfigLogger(Transform(l))
}
