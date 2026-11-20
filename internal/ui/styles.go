package ui

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

var (
	colAccent  = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7D79F6"}
	colMuted   = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#6C6C6C"}
	colBorder  = lipgloss.AdaptiveColor{Light: "#C8C8C8", Dark: "#3C3C3C"}
	colOpen    = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	colClosed  = lipgloss.AdaptiveColor{Light: "#8250DF", Dark: "#A371F7"}
	colError   = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}
	colSelBg   = lipgloss.AdaptiveColor{Light: "#E8E7FC", Dark: "#2A2950"}
	colTextDim = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#A0A0A0"}

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder)
	paneFocusedStyle = paneStyle.BorderForeground(colAccent)

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	mutedStyle  = lipgloss.NewStyle().Foreground(colMuted)
	dimStyle    = lipgloss.NewStyle().Foreground(colTextDim)
	errorStyle  = lipgloss.NewStyle().Foreground(colError).Bold(true)
	infoStyle   = lipgloss.NewStyle().Foreground(colOpen)
	selStyle    = lipgloss.NewStyle().Background(colSelBg)
	boldStyle   = lipgloss.NewStyle().Bold(true)
	openBadge   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(colOpen).Padding(0, 1).Bold(true)
	closedBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(colClosed).Padding(0, 1).Bold(true)
)

// labelChip renders a label with its GitHub colour as background and a
// readable foreground.
func labelChip(name, hex string) string {
	st := lipgloss.NewStyle().Padding(0, 1)
	if len(hex) != 6 {
		return st.Foreground(colTextDim).Render(name)
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	fg := "#FFFFFF"
	// perceived luminance
	if 0.299*float64(r)+0.587*float64(g)+0.114*float64(b) > 150 {
		fg = "#000000"
	}
	return st.Background(lipgloss.Color("#" + hex)).Foreground(lipgloss.Color(fg)).Render(name)
}
