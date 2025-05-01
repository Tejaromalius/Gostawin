package main

import (
	"fmt"
	"time"
)

// Program holds information about a tracked application window.
type Program struct {
	WMClassName string        // Unique identifier from the window manager (e.g., "Navigator.firefox")
	ProgramName string        // User-defined or default display name (e.g., "Firefox")
	Uptime      time.Duration // Accumulated duration the window has been active
}

// FormatDuration converts a time.Duration into HH:MM:SS or MM:SS string format.
func FormatDuration(d time.Duration) string {
	// Round to the nearest second
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour // Subtract hours
	m := d / time.Minute
	d -= m * time.Minute // Subtract minutes
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s) // HH:MM:SS
	}
	return fmt.Sprintf("%02d:%02d", m, s) // MM:SS
}
