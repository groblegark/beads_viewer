package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Gradients
	GradientLow  = lipgloss.Color("#44475A")
	GradientMid  = lipgloss.Color("#6272A4")
	GradientHigh = lipgloss.Color("#BD93F9")
	GradientPeak = lipgloss.Color("#FF79C6")
)

// RenderSparkline creates a textual bar chart of value (0.0 - 1.0)
func RenderSparkline(val float64, width int) string {
	chars := []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	// If width > 1, we can show history? Or just a bar?
	// Let's render a horizontal bar: `███▌    `
	
	if val < 0 { val = 0 }
	if val > 1 { val = 1 }
	
	// Calculate fullness
	fullChars := int(val * float64(width))
	remainder := (val * float64(width)) - float64(fullChars)
	
	var sb strings.Builder
	for i := 0; i < fullChars; i++ {
		sb.WriteString("█")
	}
	
	if fullChars < width {
		idx := int(remainder * float64(len(chars)))
		if idx >= len(chars) { idx = len(chars) - 1 }
		if idx > 0 {
			sb.WriteString(chars[idx])
		} else {
			sb.WriteString(" ")
		}
	}
	
	// Pad
	padding := width - fullChars - 1
	if padding > 0 {
		sb.WriteString(strings.Repeat(" ", padding))
	}
	
	return sb.String()
}

// GetHeatmapColor returns a color based on score (0-1)
func GetHeatmapColor(score float64) lipgloss.Color {
	if score > 0.8 {
		return GradientPeak
	} else if score > 0.5 {
		return GradientHigh
	} else if score > 0.2 {
		return GradientMid
	}
	return GradientLow
}
