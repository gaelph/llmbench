package tui

import (
	"context"

	"github.com/charmbracelet/bubbletea"
	"llmbench/internal/models"
)

// Messages for the TUI

// connectionTestMsg is sent when connection test completes
type connectionTestMsg struct {
	results map[string]error
}

// benchmarkProgressMsg is sent to update benchmark progress
type benchmarkProgressMsg struct {
	provider  string
	completed int
	total     int
}

// benchmarkCompleteMsg is sent when benchmark completes
type benchmarkCompleteMsg struct {
	results map[string][]models.BenchmarkResult
}

// benchmarkErrorMsg is sent when benchmark fails
type benchmarkErrorMsg struct {
	err error
}

// testConnections runs connection tests
func (m Model) testConnections() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results := m.benchmarkService.TestConnections(ctx)
		return connectionTestMsg{results: results}
	}
}

// runBenchmark runs the benchmark
func (m Model) runBenchmark() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Progress callback that sends progress updates
		progressCallback := func(provider string, completed, total int) {
			// Note: In a real implementation, you'd want to send this through a channel
			// to avoid race conditions. For simplicity, we're not doing that here.
		}
		
		results, err := m.benchmarkService.RunBenchmark(ctx, m.request, progressCallback)
		if err != nil {
			return benchmarkErrorMsg{err: err}
		}
		
		return benchmarkCompleteMsg{results: results}
	}
}
