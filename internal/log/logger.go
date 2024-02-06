// Package klog provides a Logger implementation using klog
package klog

import (
	"fmt"
	"io"

	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log"
	"k8s.io/klog"
)

// Logger is a Logger implementation using klog
type Logger struct {
}

// Infof logs an info message
func (l *Logger) Infof(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

// Errorf logs an error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	klog.Errorf(format, args...)
}

// Debugf logs a debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	klog.V(5).Infof(format, args...)
}

// Warnf logs a warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	klog.Warningf(format, args...)
}

// Failf logs a failure message
func (l *Logger) Failf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	l.Errorf("%v", err)
	return &log.LoggedError{Err: err}
}

// Printf logs a message
func (l *Logger) Printf(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(log.Level) {
}

// SetWriter sets the writer
func (l *Logger) SetWriter(w io.Writer) {
	klog.SetOutput(w)
}
