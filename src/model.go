package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// Model holds the application's state.
type Model struct {
	table          table.Model
	programs       map[string]Program // Keyed by WMClassName
	lastFetch      time.Time          // Timestamp of the last successful window fetch
	width          int                // Terminal width
	lastUpdate     time.Time          // Timestamp used for calculating uptime increments
	renaming       bool               // Flag indicating if rename input is active
	renameInput    textinput.Model    // Input field for renaming
	selectedWindow string             // WMClassName of the program being renamed
}

// refreshMsg carries the list of currently active window classes.
type refreshMsg struct {
	windows []string
}

// refreshTick triggers a data refresh cycle.
type refreshTick struct{}

// NewModel creates and initializes the application model.
func NewModel() (Model, error) {
	width, _, err := term.GetSize(os.Stdin.Fd())
	if err != nil {
		// Provide a default width if getting size fails (e.g., in non-TTY environments)
		width = 80
		log.Printf("Warning: failed to get terminal size, using default width %d: %v", width, err)
		// return Model{}, fmt.Errorf("failed to get terminal size: %w", err) // Optionally make it a hard error
	}

	programs, err := LoadPrograms()
	if err != nil {
		log.Printf("Warning: failed to load programs from cache: %v. Starting fresh.", err)
		programs = make(map[string]Program)
	}

	// Define table columns using UI constants
	columns := []table.Column{
		{Title: "Program", Width: width - UptimeColumnWidth}, // Adjusted width calculation
		{Title: "Uptime", Width: UptimeColumnWidth},
	}

	// Calculate initial table height dynamically or use a minimum
	// terminalHeight, _, _ := term.GetSize(os.Stdin.Fd()) // Could get height too
	tableHeight := MinTableHeight // Use MinTableHeight for simplicity for now

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}), // Start with empty rows
		table.WithHeight(tableHeight),
		table.WithFocused(true),
		table.WithStyles(TableStyles), // Use styles from ui.go
	)

	// Initialize text input for renaming
	input := textinput.New()
	input.Placeholder = "Enter new name"
	input.CharLimit = 156 // Example limit
	input.Width = 20      // Example width

	model := Model{
		table:       t,
		programs:    programs,
		width:       width,
		lastUpdate:  time.Now(), // Initialize last update time
		renameInput: input,
	}
	model.updateTable() // Populate table initially if data was loaded

	return model, nil
}

// Init initializes the model, triggering the first data fetch.
func (m Model) Init() tea.Cmd {
	return m.refreshData()
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.renaming {
			return m.handleRenameInput(msg)
		}
		return m.handleTableInput(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		// Optionally adjust height: m.height = msg.Height
		m.updateColumnWidths()
		// Optional: Adjust table height based on terminal height
		// m.table.SetHeight(calculateDynamicHeight(msg.Height))
		return m, nil

	case refreshMsg:
		m.updatePrograms(msg.windows) // Update program data based on fetched windows
		m.lastFetch = time.Now()
		m.updateTable() // Refresh the table view
		// Schedule the next refresh tick
		return m, tea.Tick(RefreshInterval, func(t time.Time) tea.Msg {
			return refreshTick{}
		})

	case refreshTick:
		// Trigger fetching new window data
		return m, m.refreshData()
	}

	// Delegate non-handled messages to the table component if not renaming
	if !m.renaming {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

// handleRenameInput processes key presses when the rename input is active.
func (m *Model) handleRenameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "enter":
		newName := m.renameInput.Value()
		if m.selectedWindow != "" && newName != "" {
			if prog, exists := m.programs[m.selectedWindow]; exists {
				prog.ProgramName = newName
				m.programs[m.selectedWindow] = prog
				m.updateTable() // Update table with the new name
				if err := SavePrograms(m.programs); err != nil {
					log.Printf("Error saving renamed program: %v", err)
				}
			}
		}
		m.renaming = false
		m.renameInput.Blur()
		m.renameInput.Reset()
		m.table.Focus() // Return focus to table
		return m, nil   // No further command needed here

	case "esc":
		m.renaming = false
		m.renameInput.Blur()
		m.renameInput.Reset()
		m.table.Focus() // Return focus to table
		return m, nil

	default:
		// Update the text input field
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

// handleTableInput processes key presses when the table is active.
func (m *Model) handleTableInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r", "R":
		return m, m.refreshData() // Trigger manual refresh
	case "enter":
		// Start renaming the selected program
		if len(m.table.Rows()) > 0 && m.table.Cursor() < len(m.table.Rows()) {
			selectedRow := m.table.Rows()[m.table.Cursor()]
			if len(selectedRow) > 1 { // Ensure row has expected columns
				programName := selectedRow[0]
				uptimeStr := selectedRow[1]

				// Find the corresponding program in the map
				// Note: This relies on ProgramName and formatted Uptime being unique enough for lookup.
				// A more robust approach might involve storing the WMClassName in the table row (hidden column?)
				// or searching differently.
				for wmClass, prog := range m.programs {
					if prog.ProgramName == programName && FormatDuration(prog.Uptime) == uptimeStr {
						m.selectedWindow = wmClass
						m.renameInput.SetValue(prog.ProgramName)
						m.renameInput.Focus()
						m.renaming = true
						m.table.Blur()            // Unfocus the table
						return m, textinput.Blink // Start the cursor blinking
					}
				}
				log.Printf("Warning: Could not find program '%s' with uptime '%s' to rename.", programName, uptimeStr)
			}
		}
		return m, nil // No action if no valid row selected
	case "esc":
		// Toggle focus (less useful now with dedicated rename mode)
		if m.table.Focused() {
			m.table.Blur()
		} else {
			m.table.Focus()
		}
		return m, nil
	}
	// Default: Update the table (for navigation like up/down keys)
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the UI based on the current model state.
func (m Model) View() string {
	if m.renaming {
		// Display the renaming input prompt
		programToRename := "unknown"
		if prog, exists := m.programs[m.selectedWindow]; exists {
			programToRename = prog.ProgramName
		}
		prompt := fmt.Sprintf("Rename '%s' to:\n%s\n\n(Enter to confirm, Esc to cancel)",
			programToRename,
			m.renameInput.View(), // Use the input's View method
		)
		// Apply styling to the prompt box
		return BaseStyle.Width(m.width - 2). // Adjust width dynamically
							Align(lipgloss.Center).
							Render(prompt)
	}

	// Display the main table view
	tableStr := BaseStyle.Render(m.table.View())
	// Display footer information
	footer := lipgloss.NewStyle().Faint(true).Render(
		fmt.Sprintf("Last update: %s | %d programs | Press 'q' to quit, Enter to rename, 'r' to refresh",
			m.lastFetch.Format("15:04:05"),
			len(m.programs)), // Use the actual count from the map
	)
	return tableStr + "\n" + footer
}

// updateTable refreshes the data displayed in the table.
func (m *Model) updateTable() {
	// Convert map to slice for sorting
	programsList := make([]Program, 0, len(m.programs))
	for _, p := range m.programs {
		programsList = append(programsList, p)
	}

	// Sort programs by uptime (descending)
	sort.Slice(programsList, func(i, j int) bool {
		return programsList[i].Uptime > programsList[j].Uptime
	})

	// Create table rows from sorted programs
	rows := make([]table.Row, 0, len(programsList))
	for _, p := range programsList {
		rows = append(rows, table.Row{
			p.ProgramName,
			FormatDuration(p.Uptime), // Use formatter from program.go
		})
	}
	m.table.SetRows(rows)
}

// updateColumnWidths adjusts table column widths based on terminal size.
func (m *Model) updateColumnWidths() {
	programWidth := m.width - UptimeColumnWidth - 4 // Adjust for padding/borders
	if programWidth < 10 {                          // Ensure minimum width
		programWidth = 10
	}
	m.table.SetColumns([]table.Column{
		{Title: "Program", Width: programWidth},
		{Title: "Uptime", Width: UptimeColumnWidth},
	})
}

// refreshData returns a command to fetch active window data.
func (m *Model) refreshData() tea.Cmd {
	return func() tea.Msg {
		windows, err := GetActiveWindows() // Call function from wm.go
		if err != nil {
			// Handle error appropriately, maybe return an error message
			log.Printf("Error fetching windows: %v", err)
			// Optionally return a tea.Msg indicating the error to the UI
			return refreshMsg{windows: nil} // Send empty list on error? Or a specific error msg type?
		}
		return refreshMsg{windows: windows}
	}
}

// updatePrograms updates the program uptime based on currently active windows.
func (m *Model) updatePrograms(currentWindows []string) {
	now := time.Now()
	elapsed := now.Sub(m.lastUpdate) // Time since last update cycle
	if elapsed < 0 {                 // Avoid negative duration if clocks change
		elapsed = 0
	}
	m.lastUpdate = now

	currentSet := make(map[string]struct{}) // Use struct{} for set efficiency
	for _, window := range currentWindows {
		currentSet[window] = struct{}{}
	}

	// Update uptime for programs that are still active
	for key, program := range m.programs {
		if _, isActive := currentSet[key]; isActive {
			program.Uptime += elapsed
			m.programs[key] = program
		}
		// Optional: Decide if you want to remove programs that are no longer active
		// else {
		//    delete(m.programs, key)
		// }
	}

	// Add newly detected programs
	for _, window := range currentWindows {
		if _, exists := m.programs[window]; !exists {
			m.programs[window] = Program{
				WMClassName: window,
				ProgramName: window, // Default name is the window class
				Uptime:      0,      // Start with zero uptime
			}
		}
	}

	// Persist changes to cache
	if err := SavePrograms(m.programs); err != nil {
		log.Printf("Warning: failed to save programs to cache: %v", err)
	}
}
