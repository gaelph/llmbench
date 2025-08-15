package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"llmbench/internal/models"
	"llmbench/internal/service"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
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
	StateSavePrompt
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

	// Benchmark channels for continuous progress updates
	progressChan chan benchmarkProgressMsg
	resultChan   chan tea.Msg

	// Results
	summaries map[string]models.BenchmarkSummary

	// Save functionality
	saveFilename string
	saveError    error
	saveSuccess  bool

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

type saveCompleteMsg struct {
	err error
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
		// Continue listening for more progress updates
		return m, m.listenForProgress()

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

	case saveCompleteMsg:
		if msg.err != nil {
			m.saveError = msg.err
		} else {
			m.saveSuccess = true
		}
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
	case StateSavePrompt:
		return m.handleSavePromptKeys(msg)
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
	case "s":
		// Start save process
		m.state = StateSavePrompt
		m.saveFilename = ""
		m.saveError = nil
		m.saveSuccess = false
	}
	return m, nil
}

// handleSavePromptKeys handles save prompt screen
func (m Model) handleSavePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		// Cancel save and go back to results
		m.state = StateResults
		m.saveFilename = ""
		m.saveError = nil
		m.saveSuccess = false
	case "enter":
		// Save the file
		if m.saveFilename != "" {
			return m, m.saveResults()
		}
	case "backspace":
		// Remove last character
		if len(m.saveFilename) > 0 {
			m.saveFilename = m.saveFilename[:len(m.saveFilename)-1]
		}
	default:
		// Add character to filename
		if len(msg.String()) == 1 {
			m.saveFilename += msg.String()
		}
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
	case StateSavePrompt:
		return m.renderSavePrompt()
	case StateError:
		return m.renderError()
	}
	return ""
}

// Adaptive color scheme that works on both light and dark themes
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FAFAFA"}).
			Background(lipgloss.AdaptiveColor{Light: "#5A67D8", Dark: "#7D56F4"}).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#5A67D8", Dark: "#7D56F4"})

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#2D3748", Dark: "#FAFAFA"})

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#38A169", Dark: "#04B575"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#E53E3E", Dark: "#FF5F87"})

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3182CE", Dark: "#5A56E0"})

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#A0AEC0", Dark: "#874BFD"}).
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
		modelsStr := "no models"
		if len(provider.Models) > 0 {
			if len(provider.Models) == 1 {
				modelsStr = provider.Models[0]
			} else {
				modelsStr = fmt.Sprintf("%d models", len(provider.Models))
			}
		}
		b.WriteString(fmt.Sprintf("  • %s (%s)\n", provider.Name, modelsStr))
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

// saveResults saves the benchmark results to a YAML file
func (m Model) saveResults() tea.Cmd {
	return func() tea.Msg {
		// Ensure filename has .yaml extension
		filename := m.saveFilename
		if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
			filename += ".yaml"
		}

		// Create the saved results structure (same as in benchmark.go)
		savedResults := struct {
			Metadata struct {
				Timestamp string `yaml:"timestamp"`
				Version   string `yaml:"version"`
			} `yaml:"metadata"`
			Request   models.BenchmarkRequest             `yaml:"request"`
			Results   map[string][]models.BenchmarkResult `yaml:"results"`
			Summaries map[string]models.BenchmarkSummary  `yaml:"summaries"`
		}{
			Request:   m.request,
			Results:   m.benchmarkResults,
			Summaries: m.summaries,
		}

		// Set metadata
		savedResults.Metadata.Timestamp = time.Now().Format(time.RFC3339)
		savedResults.Metadata.Version = "1.0"

		// Marshal to YAML
		data, err := yaml.Marshal(savedResults)
		if err != nil {
			return saveCompleteMsg{err: fmt.Errorf("failed to marshal results: %w", err)}
		}

		// Write to file
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return saveCompleteMsg{err: fmt.Errorf("failed to write file: %w", err)}
		}

		return saveCompleteMsg{err: nil}
	}
}

// testConnections tests connections to all providers
func (m Model) testConnections() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results := m.benchmarkService.TestConnections(ctx)
		return connectionTestMsg{results: results}
	}
}

// Global channels for progress updates (workaround for BubbleTea limitations)
var (
	globalProgressChan chan benchmarkProgressMsg
	globalResultChan   chan tea.Msg
)

// runBenchmark runs the benchmark for all providers
func (m Model) runBenchmark() tea.Cmd {
	return tea.Batch(
		m.startBenchmark(),
		m.listenForUpdates(),
	)
}

// startBenchmark starts the actual benchmark execution
func (m Model) startBenchmark() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Initialize global channels
		globalProgressChan = make(chan benchmarkProgressMsg, 100)
		globalResultChan = make(chan tea.Msg, 1)
		
		// Start benchmark in goroutine
		go func() {
			defer close(globalProgressChan)
			defer close(globalResultChan)
			
			// Progress callback to send updates via global channel
			progressCallback := func(provider string, completed, total int) {
				select {
				case globalProgressChan <- benchmarkProgressMsg{
					provider:  provider,
					completed: completed,
					total:     total,
				}:
				default:
					// Channel is full, skip this update
				}
			}

			// Run the actual benchmark
			results, err := m.benchmarkService.RunBenchmark(ctx, m.request, progressCallback)
			if err != nil {
				globalResultChan <- benchmarkErrorMsg{err: err}
			} else {
				globalResultChan <- benchmarkCompleteMsg{results: results}
			}
		}()
		
		return benchmarkStartMsg{}
	}
}

// listenForUpdates continuously listens for progress updates and completion
func (m Model) listenForUpdates() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		if globalProgressChan == nil || globalResultChan == nil {
			// Channels not ready yet, keep ticking
			return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return m.listenForUpdates()()
			})()
		}
		
		// Non-blocking check for messages
		select {
		case progress, ok := <-globalProgressChan:
			if ok {
				return progress
			}
		case result, ok := <-globalResultChan:
			if ok {
				return result
			}
		default:
			// No messages available, continue ticking
		}
		
		// Continue listening by returning another tick
		return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return m.listenForUpdates()()
		})()
	})
}

// listenForProgress continues listening for progress updates
func (m Model) listenForProgress() tea.Cmd {
	return m.listenForUpdates()
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

		// Get provider names and sort them alphabetically for consistent display
		var providers []string
		for provider := range m.benchmarkProgress {
			providers = append(providers, provider)
		}
		
		// Sort providers alphabetically
		for i := 0; i < len(providers); i++ {
			for j := i + 1; j < len(providers); j++ {
				if providers[i] > providers[j] {
					providers[i], providers[j] = providers[j], providers[i]
				}
			}
		}

		// Display progress bars in sorted order
		for _, provider := range providers {
			progress := m.benchmarkProgress[provider]
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

	b.WriteString(infoStyle.Render("Press 's' to save results, 'b' or Esc to go back, q to quit"))

	return boxStyle.Render(b.String())
}

// renderSavePrompt renders the save prompt screen
func (m Model) renderSavePrompt() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Save Results"))
	b.WriteString("\n\n")

	if m.saveSuccess {
		b.WriteString(successStyle.Render("✅ Results saved successfully!"))
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render("Press any key to continue"))
	} else if m.saveError != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error saving file: %v", m.saveError)))
		b.WriteString("\n\n")
		b.WriteString("Enter filename: ")
		b.WriteString(selectedStyle.Render(m.saveFilename + "█"))
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render("Press Enter to save, Esc to cancel"))
	} else {
		b.WriteString("Enter filename to save results:")
		b.WriteString("\n\n")
		b.WriteString("Filename: ")
		b.WriteString(selectedStyle.Render(m.saveFilename + "█"))
		b.WriteString("\n\n")
		if m.saveFilename == "" {
			b.WriteString(infoStyle.Render("Type a filename and press Enter to save, Esc to cancel"))
		} else {
			// Show preview of what will be saved
			filename := m.saveFilename
			if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
				filename += ".yaml"
			}
			b.WriteString(infoStyle.Render(fmt.Sprintf("Will save to: %s", filename)))
			b.WriteString("\n")
			b.WriteString(infoStyle.Render("Press Enter to save, Esc to cancel"))
		}
	}

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
