package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetActiveWindows executes `wmctrl -lx` to find active window classes.
// It extracts the WM_CLASS value (usually the third field).
func GetActiveWindows() ([]string, error) {
	cmd := exec.Command("wmctrl", "-lx")
	output, err := cmd.Output()
	if err != nil {
		// Check if the error is because wmctrl is not installed
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("wmctrl command failed with exit code %d: %s. Is wmctrl installed and in your PATH?", exitErr.ExitCode(), exitErr.Stderr)
		}
		// Other execution errors
		return nil, fmt.Errorf("failed to execute wmctrl command: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" { // Handle case where wmctrl returns empty output
		return []string{}, nil
	}

	classes := make([]string, 0, len(lines))
	seenClasses := make(map[string]struct{}) // To store unique classes only

	for _, line := range lines {
		fields := strings.Fields(line)
		// Expecting format like: 0x04a00007  0 Navigator.firefox  hostname Title
		if len(fields) < 3 {
			// log.Printf("Skipping malformed wmctrl line: %s", line) // Optional: Log malformed lines
			continue // Skip lines that don't have enough fields
		}

		// The WM_CLASS is typically the third field (index 2)
		windowClass := fields[2]

		// We often want the application part, e.g., "firefox" from "Navigator.firefox"
		// or "code-oss" from "code-oss.Code-oss"
		// Let's take the part *after* the first dot if one exists, otherwise the whole thing.
		// If multiple dots exist (like A.B.C), this takes "B.C". Consider if `parts[0]` is better?
		// Or maybe the last part `parts[len(parts)-1]`? Let's stick to the original logic for now:
		// *Update*: The original code took the *last* part. Let's replicate that.
		if parts := strings.Split(windowClass, "."); len(parts) > 0 {
			windowClass = parts[len(parts)-1]
		}

		// Only add unique class names
		if _, seen := seenClasses[windowClass]; !seen {
			classes = append(classes, windowClass)
			seenClasses[windowClass] = struct{}{}
		}
	}

	return classes, nil
}
