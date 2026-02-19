package util

import (
	"os"
	"syscall"
)

// Colorizer manages colored CLI output.
type Colorizer struct {
	Enabled bool
}

// NewColorizer creates a new Colorizer instance, detecting if colors should be enabled.
func NewColorizer(forceEnabled bool) *Colorizer {
	enabled := forceEnabled
	if !enabled {
		// Detect if stdout is a TTY and not redirected
		// For Windows, it might need specific checks, but for Unix-like, syscall.Isatty is common.
		// For simplicity, we assume Unix-like behavior for now.
		fd := os.Stdout.Fd()
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), 0)
		enabled = errno == 0 // If it's a TTY, errno will be 0
	}
	return &Colorizer{Enabled: enabled}
}

// applyColor applies the given ANSI color code if coloring is enabled.
func (c *Colorizer) applyColor(code, text string) string {
	if !c.Enabled {
		return text
	}
	return code + text + "\033[0m"
}

// Cyan colors the text cyan.
func (c *Colorizer) Cyan(text string) string {
	return c.applyColor("\033[36m", text) // ANSI Cyan
}

// Green colors the text green.
func (c *Colorizer) Green(text string) string {
	return c.applyColor("\033[32m", text) // ANSI Green
}

// Blue colors the text blue.
func (c *Colorizer) Blue(text string) string {
	return c.applyColor("\033[34m", text) // ANSI Blue
}

// Yellow colors the text yellow.
func (c *Colorizer) Yellow(text string) string {
	return c.applyColor("\033[33m", text) // ANSI Yellow
}

// Red colors the text red.
func (c *Colorizer) Red(text string) string {
	return c.applyColor("\033[31m", text) // ANSI Red
}

// Dim dims the text.
func (c *Colorizer) Dim(text string) string {
	return c.applyColor("\033[2m", text) // ANSI Dim
}
