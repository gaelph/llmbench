package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"llmbench/internal/models"
	"llmbench/internal/service"
	"llmbench/internal/tui"

	"github.com/spf13/cobra"
)

var (
	benchmarkCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "Run benchmark tests against configured providers",
		Long: `Run benchmark tests against all configured LLM providers.
This command will send the specified number of requests to each provider
and measure response times, success rates, and token usage.`,
		RunE: runBenchmark,
	}

	// Benchmark flags
	message     string
	requests    int
	concurrent  int
	maxTokens   int
	outputJSON  bool
	interactive bool
	streaming   bool
)

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	benchmarkCmd.Flags().StringVarP(&message, "message", "m", "Hello, how are you?", "Message to send to the LLM")
	benchmarkCmd.Flags().IntVarP(&requests, "requests", "r", 0, "Number of requests to send (overrides config)")
	benchmarkCmd.Flags().IntVarP(&concurrent, "concurrent", "c", 0, "Number of concurrent requests (overrides config)")
	benchmarkCmd.Flags().IntVar(&maxTokens, "max-tokens", 100, "Maximum tokens in response")
	benchmarkCmd.Flags().BoolVar(&outputJSON, "json", false, "Output results in JSON format")
	benchmarkCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run in interactive mode with TUI")
	benchmarkCmd.Flags().BoolVarP(&streaming, "streaming", "s", false, "Enable streaming mode with TTFT and throughput metrics")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	config := configMgr.GetBenchmarkConfig()

	// Override config with command line flags if provided
	if requests > 0 {
		config.Requests = requests
	}
	if concurrent > 0 {
		config.Concurrency = concurrent
	}

	// Create benchmark service
	benchmarkService, err := service.NewBenchmarkService(config)
	if err != nil {
		return fmt.Errorf("failed to create benchmark service: %w", err)
	}

	// Create benchmark request
	benchmarkRequest := models.BenchmarkRequest{
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
		MaxTokens: maxTokens,
		Stream:    streaming,
	}

	ctx := context.Background()

	if interactive {
		// Run interactive TUI mode
		return runInteractiveBenchmark(ctx, benchmarkService, benchmarkRequest)
	}

	// Run in CLI mode
	return runCLIBenchmark(ctx, benchmarkService, benchmarkRequest)
}

func runInteractiveBenchmark(ctx context.Context, benchmarkService *service.BenchmarkService, request models.BenchmarkRequest) error {
	app := tui.NewApp(benchmarkService, request)
	return app.Run()
}

func runCLIBenchmark(ctx context.Context, benchmarkService *service.BenchmarkService, request models.BenchmarkRequest) error {
	fmt.Println("Starting benchmark...")
	fmt.Printf("Message: %s\n", message)
	fmt.Printf("Requests per provider: %d\n", configMgr.GetBenchmarkConfig().Requests)
	fmt.Printf("Concurrency: %d\n", configMgr.GetBenchmarkConfig().Concurrency)
	fmt.Println()

	// Test connections first
	fmt.Println("Testing connections...")
	connectionResults := benchmarkService.TestConnections(ctx)

	failedConnections := 0
	for provider, err := range connectionResults {
		if err != nil {
			fmt.Printf("❌ %s: %v\n", provider, err)
			failedConnections++
		} else {
			fmt.Printf("✅ %s: Connected\n", provider)
		}
	}

	if failedConnections > 0 {
		fmt.Printf("\n⚠️  %d provider(s) failed connection test\n", failedConnections)
	}
	fmt.Println()

	// Run benchmark
	fmt.Println("Running benchmark...")

	progressCallback := func(provider string, completed, total int) {
		fmt.Printf("\r%s: %d/%d completed", provider, completed, total)
		if completed == total {
			fmt.Printf(" ✅\n")
		}
	}

	results, err := benchmarkService.RunBenchmark(ctx, request, progressCallback)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	fmt.Println("\nGenerating summary...")
	summaries := benchmarkService.GenerateSummary(results)

	if outputJSON {
		return outputJSONResults(summaries, results)
	}

	return outputTextResults(summaries)
}

func outputJSONResults(summaries map[string]models.BenchmarkSummary, results map[string][]models.BenchmarkResult) error {
	output := struct {
		Summaries map[string]models.BenchmarkSummary  `json:"summaries"`
		Results   map[string][]models.BenchmarkResult `json:"results"`
	}{
		Summaries: summaries,
		Results:   results,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputTextResults(summaries map[string]models.BenchmarkSummary) error {
	fmt.Println("\n" + strings.Repeat("=", 80))
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
