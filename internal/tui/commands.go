package tui

import (
	"llmbench/internal/models"
)

// Messages for the TUI

// connectionTestMsg is sent when connection test completes
type connectionTestMsg struct {
	results map[string]error
}

// benchmarkStartMsg is sent when benchmark starts
type benchmarkStartMsg struct{}

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
