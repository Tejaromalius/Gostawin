package main

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// --- Constants ---

const (
	// RefreshInterval defines how often the window list is checked.
	RefreshInterval = 1 * time.Second
	// UptimeColumnWidth defines the fixed width for the uptime column in the table.
	UptimeColumnWidth = 15 // Reduced width slightly
	// MinTableHeight defines the minimum number of rows the table should display.
	MinTableHeight = 10
)

// --- Styles ---

var (
	// BaseStyle is the main style for the application container.
	BaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")) // Grey border

	// TableStyles defines the custom styles for the data table.
	TableStyles = func() table.Styles {
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")). // Grey border
			BorderBottom(true).                      // Add a line below the header
			Bold(true)                               // Make header bold

		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")). // Light foreground on selection
			Background(lipgloss.Color("57")).  // Purple background on selection
			Bold(false)                        // Keep selected text normal weight

		// Optional: Customize cell styles
		// s.Cell = s.Cell.Padding(0, 1)

		// Optional: Customize viewport styles (if table height exceeds available space)
		// s.Viewport = s.Viewport.BorderStyle(lipgloss.HiddenBorder())

		return s
	}()
)
