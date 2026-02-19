package util

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel defines the severity of a log message.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Logger is a simple, concurrency-safe logger.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  LogLevel
	prefix string
	color  *Colorizer // For colored output
}

// NewLogger creates a new Logger instance.
func NewLogger(out io.Writer, level LogLevel, prefix string, colorize bool) *Logger {
	var c *Colorizer
	if colorize {
		c = NewColorizer(true)
	} else {
		c = NewColorizer(false) // No color
	}
	return &Logger{
		out:    out,
		level:  level,
		prefix: prefix,
		color:  c,
	}
}

// SetLevel sets the current logging level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetColorEnabled enables or disables colored output.
func (l *Logger) SetColorEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color.Enabled = enabled
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05") // HH:MM:SS

	var coloredMsg string
	switch level {
	case LevelDebug:
		coloredMsg = l.color.Dim(fmt.Sprintf("[%s] DEBUG %s: %s", timestamp, l.prefix, msg))
	case LevelInfo:
		coloredMsg = fmt.Sprintf("[%s] INFO %s: %s", timestamp, l.prefix, msg)
	case LevelWarn:
		coloredMsg = l.color.Yellow(fmt.Sprintf("[%s] WARN %s: %s", timestamp, l.prefix, msg))
	case LevelError:
		coloredMsg = l.color.Red(fmt.Sprintf("[%s] ERROR %s: %s", timestamp, l.prefix, msg))
	case LevelFatal:
		coloredMsg = l.color.Red(fmt.Sprintf("[%s] FATAL %s: %s", timestamp, l.prefix, msg))
	default:
		coloredMsg = msg // Should not happen
	}

	_, _ = fmt.Fprintln(l.out, coloredMsg)

	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
}

// Global logger instance
var defaultLogger = NewLogger(os.Stderr, LevelInfo, "HyperWapp", true)

// exposed functions for global usage
func SetLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

func SetColorEnabled(enabled bool) {
	defaultLogger.SetColorEnabled(enabled)
}

func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a fatal message and exits.
func Fatal(format string, args ...interface{}) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05") // HH:MM:SS
	coloredMsg := defaultLogger.color.Red(fmt.Sprintf("[%s] FATAL %s: %s", timestamp, defaultLogger.prefix, msg))

	fmt.Fprintln(os.Stderr, coloredMsg) // Print directly to stderr
	os.Exit(1)
}
