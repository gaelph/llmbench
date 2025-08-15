package tui

import (
	"fmt"
	"strings"

	"llmbench/internal/models"
	"llmbench/internal/service"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// App represents the TUI application
type App struct {
	benchmarkService *service.BenchmarkService
	request          models.BenchmarkRequest
}

// NewApp creates a new TUI application
func NewApp(benchmarkService *service.BenchmarkService, request models.BenchmarkRequest) *App {
	return &App{
		benchmarkService: benchmarkService,
		request:          request,
	}
}

// Run starts the TUI application
func (a *App) Run() error {
	model := newModel(a.benchmarkService, a.request)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// State represents the current state of the application
type State int

const (
	StateMenu State = iota
	StateConnectionTest
	StateBenchmarkRunning
	StateResults
	StateError
)

// Model represents the TUI model
type Model struct {
	state            State
	benchmarkService *service.BenchmarkService
	request          models.BenchmarkRequest

	// Menu
	menuCursor int
	menuItems  []string

	// Connection test
	connectionResults map[string]error
	connectionDone    bool

	// Benchmark
	benchmarkResults  map[string][]models.BenchmarkResult
	benchmarkProgress map[string]BenchmarkProgress
	benchmarkDone     bool
	benchmarkError    error

	// Results
	summaries map[string]models.BenchmarkSummary

	// UI
	width  int
	height int
	err    error
}

// BenchmarkProgress tracks progress for each provider
type BenchmarkProgress struct {
	Completed int
	Total     int
}

// newModel creates a new model
func newModel(benchmarkService *service.BenchmarkService, request models.BenchmarkRequest) Model {
	return Model{
		state:            StateMenu,
		benchmarkService: benchmarkService,
		request:          request,
		menuItems: []string{
			"Test Connections",
			"Run Benchmark",
			"Quit",
		},
		benchmarkProgress: make(map[string]BenchmarkProgress),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case connectionTestMsg:
		m.connectionResults = msg.results
		m.connectionDone = true
		return m, nil

	case benchmarkProgressMsg:
		if m.benchmarkProgress == nil {
			m.benchmarkProgress = make(map[string]BenchmarkProgress)
		}
		m.benchmarkProgress[msg.provider] = BenchmarkProgress{
			Completed: msg.completed,
			Total:     msg.total,
		}
		return m, nil

	case benchmarkCompleteMsg:
		m.benchmarkResults = msg.results
		m.benchmarkDone = true
		m.summaries = m.benchmarkService.GenerateSummary(msg.results)
		m.state = StateResults
		return m, nil

	case benchmarkErrorMsg:
		m.benchmarkError = msg.err
		m.state = StateError
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateMenu:
		return m.handleMenuKeys(msg)
	case StateConnectionTest:
		return m.handleConnectionTestKeys(msg)
	case StateBenchmarkRunning:
		return m.handleBenchmarkKeys(msg)
	case StateResults:
		return m.handleResultsKeys(msg)
	case StateError:
		return m.handleErrorKeys(msg)
	}
	return m, nil
}

// handleMenuKeys handles menu navigation
func (m Model) handleMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}
	case "enter", " ":
		switch m.menuCursor {
		case 0: // Test Connections
			m.state = StateConnectionTest
			m.connectionDone = false
			return m, m.testConnections()
		case 1: // Run Benchmark
			m.state = StateBenchmarkRunning
			m.benchmarkDone = false
			m.benchmarkProgress = make(map[string]BenchmarkProgress)
			return m, m.runBenchmark()
		case 2: // Quit
			return m, tea.Quit
		}
	}
	return m, nil
}

// handleConnectionTestKeys handles connection test screen
func (m Model) handleConnectionTestKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		if m.connectionDone {
			m.state = StateMenu
		}
	}
	return m, nil
}

// handleBenchmarkKeys handles benchmark screen
func (m Model) handleBenchmarkKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	return m, nil
}

// handleResultsKeys handles results screen
func (m Model) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.state = StateMenu
	}
	return m, nil
}

// handleErrorKeys handles error screen
func (m Model) handleErrorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.state = StateMenu
	}
	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.state {
	case StateMenu:
		return m.renderMenu()
	case StateConnectionTest:
		return m.renderConnectionTest()
	case StateBenchmarkRunning:
		return m.renderBenchmark()
	case StateResults:
		return m.renderResults()
	case StateError:
		return m.renderError()
	}
	return ""
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5A56E0"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)
)

// renderMenu renders the main menu
func (m Model) renderMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("LLM Benchmark Tool"))
	b.WriteString("\n\n")

	providers := m.benchmarkService.GetProviders()
	b.WriteString(fmt.Sprintf("Configured providers: %d\n", len(providers)))
	for _, provider := range providers {
		b.WriteString(fmt.Sprintf("  • %s (%s)\n", provider.Name, provider.Model))
	}
	b.WriteString("\n")

	b.WriteString("Choose an option:\n\n")

	for i, item := range m.menuItems {
		cursor := " "
		if m.menuCursor == i {
			cursor = ">"
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%s %s", cursor, item)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("%s %s", cursor, item)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(infoStyle.Render("Use ↑/↓ to navigate, Enter to select, q to quit"))

	return boxStyle.Render(b.String())
}

// renderConnectionTest renders the connection test screen
func (m Model) renderConnectionTest() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Connection Test"))
	b.WriteString("\n\n")

	if !m.connectionDone {
		b.WriteString("Testing connections to providers...\n\n")
		b.WriteString("⏳ Please wait...")
	} else {
		b.WriteString("Connection test results:\n\n")

		successCount := 0
		for provider, err := range m.connectionResults {
			if err != nil {
				b.WriteString(errorStyle.Render(fmt.Sprintf("❌ %s: %v", provider, err)))
			} else {
				b.WriteString(successStyle.Render(fmt.Sprintf("✅ %s: Connected", provider)))
				successCount++
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		totalCount := len(m.connectionResults)
		if successCount == totalCount {
			b.WriteString(successStyle.Render(fmt.Sprintf("🎉 All %d providers connected successfully!", totalCount)))
		} else {
			b.WriteString(errorStyle.Render(fmt.Sprintf("⚠️  %d/%d providers failed connection test", totalCount-successCount, totalCount)))
		}

		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render("Press 'b' or Esc to go back, q to quit"))
	}

	return boxStyle.Render(b.String())
}

// renderBenchmark renders the benchmark running screen
func (m Model) renderBenchmark() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Running Benchmark"))
	b.WriteString("\n\n")

	if !m.benchmarkDone {
		b.WriteString("Benchmark in progress...\n\n")

		for provider, progress := range m.benchmarkProgress {
			percentage := float64(progress.Completed) / float64(progress.Total) * 100
			b.WriteString(fmt.Sprintf("%s: %d/%d (%.1f%%)\n", provider, progress.Completed, progress.Total, percentage))

			// Simple progress bar
			barWidth := 30
			filled := int(float64(barWidth) * percentage / 100)
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			b.WriteString(fmt.Sprintf("[%s]\n\n", bar))
		}

		b.WriteString(infoStyle.Render("Press Ctrl+C to cancel"))
	}

	return boxStyle.Render(b.String())
}

// renderResults renders the results screen
func (m Model) renderResults() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Benchmark Results"))
	b.WriteString("\n\n")

	for provider, summary := range m.summaries {
		b.WriteString(fmt.Sprintf("📊 %s\n", strings.ToUpper(provider)))
		b.WriteString(strings.Repeat("-", 30) + "\n")
		b.WriteString(fmt.Sprintf("Total Requests:     %d\n", summary.TotalRequests))
		b.WriteString(fmt.Sprintf("Successful:         %d\n", summary.SuccessfulReqs))
		b.WriteString(fmt.Sprintf("Failed:             %d\n", summary.FailedRequests))
		b.WriteString(fmt.Sprintf("Error Rate:         %.2f%%\n", summary.ErrorRate))
		b.WriteString(fmt.Sprintf("Avg Response Time:  %v\n", summary.AvgResponseTime))
		b.WriteString(fmt.Sprintf("Min Response Time:  %v\n", summary.MinResponseTime))
		b.WriteString(fmt.Sprintf("Max Response Time:  %v\n", summary.MaxResponseTime))
		b.WriteString(fmt.Sprintf("Total Tokens:       %d\n", summary.TotalTokens))
		b.WriteString("\n")
	}

	b.WriteString(infoStyle.Render("Press 'b' or Esc to go back, q to quit"))

	return boxStyle.Render(b.String())
}

// renderError renders the error screen
func (m Model) renderError() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Error"))
	b.WriteString("\n\n")

	b.WriteString(errorStyle.Render(fmt.Sprintf("❌ %v", m.benchmarkError)))
	b.WriteString("\n\n")
	b.WriteString(infoStyle.Render("Press 'b' or Esc to go back, q to quit"))

	return boxStyle.Render(b.String())
}
