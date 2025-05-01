package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Basic command-line flag handling (kept simple)
	if len(os.Args) > 1 && (os.Args[1] == "-u" || os.Args[1] == "--update") {
		fmt.Println("Update mode detected (manual parsing) - Note: This refactored version doesn't use this flag actively.")
	}

	// Create the initial model
	m, err := NewModel()
	if err != nil {
		fmt.Printf("Error initializing application: %v\n", err)
		os.Exit(1)
	}

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
