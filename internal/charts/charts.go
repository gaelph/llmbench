package charts

import (
	"fmt"
	"sort"
	"strings"

	"llmbench/internal/models"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"
)

// ChartGenerator handles the generation of charts for benchmark results
type ChartGenerator struct {
	width  int
	height int
}

// NewChartGenerator creates a new chart generator with specified dimensions
func NewChartGenerator(width, height int) *ChartGenerator {
	return &ChartGenerator{
		width:  width,
		height: height,
	}
}

// LegendEntry represents a single entry in the chart legend
type LegendEntry struct {
	Label string
	Value float64
	Unit  string
	Color string
}

// generateLegend creates a formatted legend showing the numerical values
func (cg *ChartGenerator) generateLegend(entries []LegendEntry, title string) string {
	if len(entries) == 0 {
		return ""
	}

	// Sort entries by value (descending) for better readability
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value > entries[j].Value
	})

	var legend strings.Builder
	legend.WriteString(fmt.Sprintf("\n📋 %s Legend:\n", title))
	legend.WriteString(strings.Repeat("─", cg.width) + "\n")

	// Find the longest label for alignment
	maxLabelLen := 0
	for _, entry := range entries {
		if len(entry.Label) > maxLabelLen {
			maxLabelLen = len(entry.Label)
		}
	}

	// Generate legend entries with proper alignment
	for i, entry := range entries {
		// Create colored indicator
		colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(entry.Color))
		indicator := colorStyle.Render("■")
		
		// Format the value with appropriate precision
		var valueStr string
		if entry.Value < 1 {
			valueStr = fmt.Sprintf("%.3f", entry.Value)
		} else if entry.Value < 10 {
			valueStr = fmt.Sprintf("%.2f", entry.Value)
		} else {
			valueStr = fmt.Sprintf("%.1f", entry.Value)
		}

		// Pad label for alignment
		paddedLabel := fmt.Sprintf("%-*s", maxLabelLen, entry.Label)
		
		legend.WriteString(fmt.Sprintf("  %s %s: %s %s\n", 
			indicator, paddedLabel, valueStr, entry.Unit))
		
		// Add separator line between entries (except for the last one)
		if i < len(entries)-1 {
			legend.WriteString("    " + strings.Repeat("·", maxLabelLen+10) + "\n")
		}
	}

	return legend.String()
}

// GenerateTTFTChart creates a bar chart showing Time to First Token for each model
func (cg *ChartGenerator) GenerateTTFTChart(summaries map[string]models.BenchmarkSummary) string {
	if len(summaries) == 0 {
		return "No data available for TTFT chart"
	}

	// Filter and sort keys to ensure consistent ordering
	var validKeys []string
	for key, summary := range summaries {
		if summary.IsStreaming && summary.AvgTimeToFirstToken > 0 {
			validKeys = append(validKeys, key)
		}
	}
	
	if len(validKeys) == 0 {
		return "No streaming data available for TTFT chart"
	}
	
	sort.Strings(validKeys) // Ensure consistent ordering

	var barData []barchart.BarData
	var legendEntries []LegendEntry
	colors := []string{"10", "9", "11", "12", "13", "14", "15", "6"} // Green, Red, Yellow, Blue, Magenta, Cyan, White, Cyan

	for i, key := range validKeys {
		summary := summaries[key]
		// Convert duration to milliseconds for better readability
		ttftMs := float64(summary.AvgTimeToFirstToken.Nanoseconds()) / 1e6
		
		color := colors[i%len(colors)]
		
		barData = append(barData, barchart.BarData{
			Label: key,
			Values: []barchart.BarValue{
				{Name: "TTFT", Value: ttftMs, Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
			},
		})

		// Add to legend
		legendEntries = append(legendEntries, LegendEntry{
			Label: key,
			Value: ttftMs,
			Unit:  "ms",
			Color: color,
		})
	}

	bc := barchart.New(cg.width, cg.height)
	bc.PushAll(barData)
	bc.Draw()

	// Generate chart with legend
	result := fmt.Sprintf("📊 Time to First Token (ms)\n%s\n%s", 
		strings.Repeat("─", cg.width), bc.View())
	
	// Add legend
	legend := cg.generateLegend(legendEntries, "TTFT Values")
	result += legend

	return result
}

// GenerateThroughputChart creates a bar chart showing token throughput for each model
func (cg *ChartGenerator) GenerateThroughputChart(summaries map[string]models.BenchmarkSummary) string {
	if len(summaries) == 0 {
		return "No data available for throughput chart"
	}

	// Filter and sort keys to ensure consistent ordering
	var validKeys []string
	for key, summary := range summaries {
		if summary.IsStreaming && summary.AvgTokenThroughput > 0 {
			validKeys = append(validKeys, key)
		}
	}
	
	if len(validKeys) == 0 {
		return "No streaming data available for throughput chart"
	}
	
	sort.Strings(validKeys) // Ensure consistent ordering

	var barData []barchart.BarData
	var legendEntries []LegendEntry
	colors := []string{"10", "9", "11", "12", "13", "14", "15", "6"} // Green, Red, Yellow, Blue, Magenta, Cyan, White, Cyan

	for i, key := range validKeys {
		summary := summaries[key]
		color := colors[i%len(colors)]
		
		barData = append(barData, barchart.BarData{
			Label: key,
			Values: []barchart.BarValue{
				{Name: "Throughput", Value: summary.AvgTokenThroughput, Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
			},
		})

		// Add to legend
		legendEntries = append(legendEntries, LegendEntry{
			Label: key,
			Value: summary.AvgTokenThroughput,
			Unit:  "tokens/sec",
			Color: color,
		})
	}

	bc := barchart.New(cg.width, cg.height)
	bc.PushAll(barData)
	bc.Draw()

	// Generate chart with legend
	result := fmt.Sprintf("📊 Token Throughput (tokens/sec)\n%s\n%s", 
		strings.Repeat("─", cg.width), bc.View())
	
	// Add legend
	legend := cg.generateLegend(legendEntries, "Throughput Values")
	result += legend

	return result
}

// GenerateResponseTimeChart creates a bar chart showing average response times for each model
func (cg *ChartGenerator) GenerateResponseTimeChart(summaries map[string]models.BenchmarkSummary) string {
	if len(summaries) == 0 {
		return "No data available for response time chart"
	}

	// Filter and sort keys to ensure consistent ordering
	var validKeys []string
	for key, summary := range summaries {
		if summary.AvgResponseTime > 0 {
			validKeys = append(validKeys, key)
		}
	}
	
	if len(validKeys) == 0 {
		return "No data available for response time chart"
	}
	
	sort.Strings(validKeys) // Ensure consistent ordering

	var barData []barchart.BarData
	var legendEntries []LegendEntry
	colors := []string{"10", "9", "11", "12", "13", "14", "15", "6"} // Green, Red, Yellow, Blue, Magenta, Cyan, White, Cyan

	for i, key := range validKeys {
		summary := summaries[key]
		// Convert duration to milliseconds for better readability
		responseTimeMs := float64(summary.AvgResponseTime.Nanoseconds()) / 1e6
		
		color := colors[i%len(colors)]
		
		barData = append(barData, barchart.BarData{
			Label: key,
			Values: []barchart.BarValue{
				{Name: "Response Time", Value: responseTimeMs, Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
			},
		})

		// Add to legend
		legendEntries = append(legendEntries, LegendEntry{
			Label: key,
			Value: responseTimeMs,
			Unit:  "ms",
			Color: color,
		})
	}

	bc := barchart.New(cg.width, cg.height)
	bc.PushAll(barData)
	bc.Draw()

	// Generate chart with legend
	result := fmt.Sprintf("📊 Average Response Time (ms)\n%s\n%s", 
		strings.Repeat("─", cg.width), bc.View())
	
	// Add legend
	legend := cg.generateLegend(legendEntries, "Response Time Values")
	result += legend

	return result
}

// GenerateAllCharts generates all available charts for the given summaries
func (cg *ChartGenerator) GenerateAllCharts(summaries map[string]models.BenchmarkSummary) string {
	var result string
	
	// Check if we have any streaming data
	hasStreamingData := false
	for _, summary := range summaries {
		if summary.IsStreaming {
			hasStreamingData = true
			break
		}
	}

	// Generate response time chart (always available)
	result += cg.GenerateResponseTimeChart(summaries) + "\n\n"

	// Generate streaming-specific charts if we have streaming data
	if hasStreamingData {
		result += cg.GenerateTTFTChart(summaries) + "\n\n"
		result += cg.GenerateThroughputChart(summaries) + "\n\n"
	}

	return result
}
