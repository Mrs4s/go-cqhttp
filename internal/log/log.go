// Package log implements a simple logging.
package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Entry represents a log entry.
type Entry struct {
	Level   Level
	Time    time.Time
	Message string
}

// Formatter is log formatter interface.
type Formatter interface {
	Format(entry *Entry) []byte
}

// Hook is log hook interface.
type Hook interface {
	Level() Level
	Fire(entry *Entry)
}

// Logger is a simple logger.
type Logger struct {
	mu        sync.RWMutex
	Level     Level
	Formatter Formatter
	Output    io.Writer

	hookLevel Level
	hooker    Hook
}

var std = &Logger{
	Level:  InfoLevel,
	Output: os.Stderr,
}

func (l *Logger) log(level Level, message string) {
	entry := &Entry{
		Level:   level,
		Time:    time.Now(),
		Message: message,
	}

	if l.hooker != nil && level < l.hookLevel {
		l.hooker.Fire(entry)
	}

	_, _ = l.Output.Write(l.Formatter.Format(entry))
}

// Logf ...
func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if level < l.Level {
		return
	}
	l.log(level, fmt.Sprintf(format, args...))
}

// Log ...
func (l *Logger) Log(level Level, a ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if level < l.Level {
		return
	}
	l.log(level, fmt.Sprint(a...))
}

// Fatalf ...
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logf(FatalLevel, format, args...)
	os.Exit(1)
}

// Debugf ...
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logf(DebugLevel, format, args...)
}

// Errorf ...
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(ErrorLevel, format, args...)
}

// Warnf ...
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logf(WarnLevel, format, args...)
}

// Infof ...
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logf(InfoLevel, format, args...)
}

// Fatal ...
func (l *Logger) Fatal(a ...interface{}) {
	l.Log(FatalLevel, a...)
	os.Exit(1)
}

// Error ...
func (l *Logger) Error(a ...interface{}) {
	l.Log(ErrorLevel, a...)
}

// Warn ...
func (l *Logger) Warn(a ...interface{}) {
	l.Log(WarnLevel, a...)
}

// Info ...
func (l *Logger) Info(a ...interface{}) {
	l.Log(InfoLevel, a...)
}

// Debug ...
func (l *Logger) Debug(a ...interface{}) {
	l.Log(DebugLevel, a...)
}

// Fatalf ...
func Fatalf(format string, args ...interface{}) {
	std.Fatalf(format, args...)
}

// Errorf ...
func Errorf(format string, args ...interface{}) {
	std.Errorf(format, args...)
}

// Warnf ...
func Warnf(format string, args ...interface{}) {
	std.Warnf(format, args...)
}

// Infof ...
func Infof(format string, args ...interface{}) {
	std.Infof(format, args...)
}

// Debugf ...
func Debugf(format string, args ...interface{}) {
	std.Debugf(format, args...)
}

// Fatal ...
func Fatal(a ...interface{}) {
	std.Fatal(a...)
}

// Error ...
func Error(a ...interface{}) {
	std.Error(a...)
}

// Warn ...
func Warn(a ...interface{}) {
	std.Warn(a...)
}

// Info ...
func Info(a ...interface{}) {
	std.Info(a...)
}

// Debug ...
func Debug(a ...interface{}) {
	std.Debug(a...)
}

// SetFormatter set formatter for std logger
func SetFormatter(f Formatter) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.Formatter = f
}

// SetOutput set output for std logger
func SetOutput(w io.Writer) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.Output = w
}

// SetLevel set level for std logger
func SetLevel(l Level) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.Level = l
}

// SetHook set hooker for std logger
func SetHook(h Hook) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.hooker = h
	std.hookLevel = h.Level()
}
