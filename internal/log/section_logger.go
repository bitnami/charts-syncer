package klog

import (
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log"
	"k8s.io/klog"
)

// SectionLogger is a SectionLogger implementation using klog
type SectionLogger struct {
	Logger
}

// ExecuteStep executes a function while showing an indeterminate progress animation
func (l *SectionLogger) ExecuteStep(title string, fn func() error) error {
	klog.Infof(title)
	return fn()
}

// PrefixText returns the indented version of the provided text
func (l *SectionLogger) PrefixText(txt string) string {
	return txt
}

// StartSection starts a new log section
func (l *SectionLogger) StartSection(string) log.SectionLogger {
	return l
}

// ProgressBar returns a new  progress bar
func (l *SectionLogger) ProgressBar() log.ProgressBar {
	return log.NewLoggedProgressBar(&Logger{})
}

// Successf logs a new success message (more efusive than Infof)
func (l *SectionLogger) Successf(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

// Section executes the provided function inside a new section
func (l *SectionLogger) Section(title string, fn func(log.SectionLogger) error) error {
	klog.Infof(title)
	return fn(l)
}

// NewKlogSectionLogger returns a new SectionLogger implemented by klog
func NewKlogSectionLogger() *SectionLogger {
	return &SectionLogger{Logger: Logger{}}
}
