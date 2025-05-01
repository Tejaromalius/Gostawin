package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const (
	refreshInterval   = 1 * time.Second
	uptimeColumnWidth = 30
	minTableHeight    = 10
)

type Program struct {
	WMClassName string
	ProgramName string
	Uptime      time.Duration
}

type Model struct {
	table          table.Model
	programs       map[string]Program
	lastFetch      time.Time
	width          int
	lastUpdate     time.Time
	renaming       bool
	renameInput    textinput.Model
	selectedWindow string
}

type refreshMsg struct {
	windows []string
}

type refreshTick struct{}

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	tableStyles = func() table.Styles {
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		return s
	}()
)

func getCachePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".cache", "gostawin", "stats.gob"), nil
}

func ensureCacheDir() error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(cachePath), 0o755)
}

func savePrograms(programs map[string]Program) error {
	if err := ensureCacheDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(programs); err != nil {
		return fmt.Errorf("failed to encode programs: %w", err)
	}

	return nil
}

func loadPrograms() (map[string]Program, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Program), nil
		}
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	var programs map[string]Program
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&programs); err != nil {
		return nil, fmt.Errorf("failed to decode programs: %w", err)
	}

	return programs, nil
}

func NewModel() (Model, error) {
	width, _, err := term.GetSize(os.Stdin.Fd())
	if err != nil {
		return Model{}, fmt.Errorf("failed to get terminal size: %w", err)
	}

	programs, err := loadPrograms()
	if err != nil {
		log.Printf("Warning: failed to load programs from cache: %v", err)
		programs = make(map[string]Program)
	}

	columns := []table.Column{
		{Title: "Program", Width: width - uptimeColumnWidth},
		{Title: "Uptime", Width: uptimeColumnWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(minTableHeight),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles)

	input := textinput.New()
	input.Placeholder = "Enter new name"
	input.Focus()

	return Model{
		table:       t,
		programs:    programs,
		width:       width,
		lastUpdate:  time.Now(),
		renameInput: input,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.refreshData()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.renaming {
			switch msg.String() {
			case "enter":
				if m.selectedWindow != "" && m.renameInput.Value() != "" {
					if prog, exists := m.programs[m.selectedWindow]; exists {
						prog.ProgramName = m.renameInput.Value()
						m.programs[m.selectedWindow] = prog
						m.updateTable()
						if err := savePrograms(m.programs); err != nil {
							log.Printf("Error saving programs: %v", err)
						}
					}
				}
				m.renaming = false
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			case "esc":
				m.renaming = false
				m.renameInput.Reset()
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		} else {
			switch msg.String() {
			case "esc":
				if m.table.Focused() {
					m.table.Blur()
				} else {
					m.table.Focus()
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			case "r", "R":
				return m, m.refreshData()
			case "enter":
				if len(m.table.Rows()) > 0 && m.table.Cursor() < len(m.table.Rows()) {
					selectedRow := m.table.Rows()[m.table.Cursor()]
					if len(selectedRow) > 0 {
						for wmClass, prog := range m.programs {
							if prog.ProgramName == selectedRow[0] && formatDuration(prog.Uptime) == selectedRow[1] {
								m.selectedWindow = wmClass
								m.renameInput.SetValue(prog.ProgramName)
								m.renaming = true
								break
							}
						}
					}
				}
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.updateColumns()
	case refreshMsg:
		m.updatePrograms(msg.windows)
		m.lastFetch = time.Now()
		m.updateTable()
		return m, tea.Tick(refreshInterval, func(time.Time) tea.Msg {
			return refreshTick{}
		})
	case refreshTick:
		return m, m.refreshData()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.renaming {
		return baseStyle.Render(
			fmt.Sprintf("Rename '%s' to:\n> %s\n\n(Press Enter to confirm, Esc to cancel)",
				m.programs[m.selectedWindow].ProgramName,
				m.renameInput.Value()),
		)
	}

	return baseStyle.Render(m.table.View()) + "\n" +
		lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf("Last updated: %s | %d programs | Press 'q' to quit, Enter to rename",
				m.lastFetch.Format("15:04:05"),
				len(m.programs)),
		)
}

func (m *Model) updateTable() {
	programs := make([]Program, 0, len(m.programs))
	for _, p := range m.programs {
		programs = append(programs, p)
	}

	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Uptime > programs[j].Uptime
	})

	rows := make([]table.Row, 0, len(programs))
	for _, p := range programs {
		rows = append(rows, table.Row{
			p.ProgramName,
			formatDuration(p.Uptime),
		})
	}
	m.table.SetRows(rows)
}

func (m *Model) updateColumns() {
	columns := []table.Column{
		{Title: "Program", Width: m.width - uptimeColumnWidth},
		{Title: "Uptime", Width: uptimeColumnWidth},
	}
	m.table.SetColumns(columns)
}

func (m *Model) refreshData() tea.Cmd {
	return func() tea.Msg {
		windows, err := getActiveWindows()
		if err != nil {
			log.Printf("Error fetching windows: %v", err)
			return nil
		}
		return refreshMsg{windows: windows}
	}
}

func (m *Model) updatePrograms(currentWindows []string) {
	now := time.Now()
	elapsed := now.Sub(m.lastUpdate)
	m.lastUpdate = now

	currentSet := make(map[string]bool)
	for _, window := range currentWindows {
		currentSet[window] = true
	}

	for key, program := range m.programs {
		if currentSet[key] {
			program.Uptime += elapsed
			m.programs[key] = program
		}
	}

	for _, window := range currentWindows {
		if _, exists := m.programs[window]; !exists {
			m.programs[window] = Program{
				WMClassName: window,
				ProgramName: window, // Default name is the window class
				Uptime:      0,
			}
		}
	}

	if err := savePrograms(m.programs); err != nil {
		log.Printf("Warning: failed to save programs to cache: %v", err)
	}
}

func getActiveWindows() ([]string, error) {
	cmd := exec.Command("wmctrl", "-lx")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wmctrl command failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	classes := make([]string, 0, len(lines))

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Get the window class (third field)
		windowClass := fields[2]
		// Split on '.' and take the last part if it exists
		if parts := strings.Split(windowClass, "."); len(parts) > 0 {
			windowClass = parts[len(parts)-1]
		}
		classes = append(classes, windowClass)
	}
	return classes, nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-u" || os.Args[1] == "--update") {
		fmt.Println("Update mode detected (manual parsing)!")
	}

	m, err := NewModel()
	if err != nil {
		fmt.Printf("Error initializing application: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
