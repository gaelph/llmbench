package cmd

import (
	"fmt"
	"os"
	"strings"

	"llmbench/internal/charts"
	"llmbench/internal/models"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	displayCmd = &cobra.Command{
		Use:   "display <results-file>",
		Short: "Display saved benchmark results",
		Long: `Display benchmark results from a previously saved YAML file.
This command allows you to view results from past benchmark runs without
re-running the benchmark. You can display either text summary or charts.`,
		Args: cobra.ExactArgs(1),
		RunE: runDisplay,
	}

	// Display flags
	displayCharts bool
	displayJSON   bool
)

func init() {
	rootCmd.AddCommand(displayCmd)

	displayCmd.Flags().BoolVar(&displayCharts, "charts", false, "Display bar charts for TTFT and throughput metrics")
	displayCmd.Flags().BoolVar(&displayJSON, "json", false, "Output results in JSON format")
}

func runDisplay(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Load benchmark results from YAML file
	resultsFile, err := loadBenchmarkResults(filename)
	if err != nil {
		return fmt.Errorf("failed to load results from %s: %w", filename, err)
	}

	// Display file metadata
	fmt.Printf("📁 Loaded results from: %s\n", filename)
	fmt.Printf("🕒 Benchmark run time: %s\n", resultsFile.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("💬 Message: %s\n", resultsFile.Metadata.Message)
	fmt.Printf("📊 Requests: %d, Concurrency: %d, Max Tokens: %d\n", 
		resultsFile.Metadata.Requests, resultsFile.Metadata.Concurrency, resultsFile.Metadata.MaxTokens)
	if resultsFile.Metadata.Streaming {
		fmt.Printf("🚀 Streaming: enabled\n")
	}
	fmt.Println()

	if displayJSON {
		return outputJSONResults(resultsFile.Summaries, resultsFile.Results)
	}

	return displayTextResults(resultsFile.Summaries)
}

func displayTextResults(summaries map[string]models.BenchmarkSummary) error {
	// If charts are requested, show only charts
	if displayCharts {
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("BENCHMARK CHARTS")
		fmt.Println(strings.Repeat("=", 80))
		
		// Create chart generator with appropriate dimensions
		chartGen := charts.NewChartGenerator(60, 15)
		chartsOutput := chartGen.GenerateAllCharts(summaries)
		fmt.Print(chartsOutput)
		fmt.Println(strings.Repeat("=", 80))
		return nil
	}

	// Otherwise, show text summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	for _, summary := range summaries {
		// Display provider and model name clearly
		if summary.ModelName != "" {
			fmt.Printf("\n📊 %s - %s\n", strings.ToUpper(summary.Provider), summary.ModelName)
		} else {
			fmt.Printf("\n📊 %s\n", strings.ToUpper(summary.Provider))
		}
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Total Requests:     %d\n", summary.TotalRequests)
		fmt.Printf("Successful:         %d\n", summary.SuccessfulReqs)
		fmt.Printf("Failed:             %d\n", summary.FailedRequests)
		fmt.Printf("Error Rate:         %.2f%%\n", summary.ErrorRate)
		fmt.Printf("Avg Response Time:  %v\n", summary.AvgResponseTime)
		fmt.Printf("Min Response Time:  %v\n", summary.MinResponseTime)
		fmt.Printf("Max Response Time:  %v\n", summary.MaxResponseTime)
		fmt.Printf("Total Tokens:       %d\n", summary.TotalTokens)
		
		// Display streaming metrics if available
		if summary.IsStreaming {
			fmt.Println("\n🚀 STREAMING METRICS")
			fmt.Println(strings.Repeat("-", 20))
			fmt.Printf("Avg Time to First Token: %v\n", summary.AvgTimeToFirstToken)
			fmt.Printf("Min Time to First Token: %v\n", summary.MinTimeToFirstToken)
			fmt.Printf("Max Time to First Token: %v\n", summary.MaxTimeToFirstToken)
			fmt.Printf("Avg Token Throughput:    %.2f tokens/sec\n", summary.AvgTokenThroughput)
			fmt.Printf("Min Token Throughput:    %.2f tokens/sec\n", summary.MinTokenThroughput)
			fmt.Printf("Max Token Throughput:    %.2f tokens/sec\n", summary.MaxTokenThroughput)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	return nil
}

// loadBenchmarkResults loads benchmark results from a YAML file
func loadBenchmarkResults(filename string) (*BenchmarkResultsFile, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal YAML
	var resultsFile BenchmarkResultsFile
	if err := yaml.Unmarshal(data, &resultsFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &resultsFile, nil
}
