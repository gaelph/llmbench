package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"llmbench/internal/service"
)

var (
	testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test connections to configured providers",
		Long: `Test connectivity to all configured LLM providers.
This command sends a simple test message to verify that the providers
are reachable and responding correctly.`,
		RunE: runTest,
	}
)

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	config := configMgr.GetBenchmarkConfig()

	// Create benchmark service
	benchmarkService, err := service.NewBenchmarkService(config)
	if err != nil {
		return fmt.Errorf("failed to create benchmark service: %w", err)
	}

	fmt.Println("Testing connections to configured providers...")
	fmt.Println()

	ctx := context.Background()
	results := benchmarkService.TestConnections(ctx)

	successCount := 0
	totalCount := len(results)

	for provider, err := range results {
		if err != nil {
			fmt.Printf("❌ %s: %v\n", provider, err)
		} else {
			fmt.Printf("✅ %s: Connection successful\n", provider)
			successCount++
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d/%d providers connected successfully\n", successCount, totalCount)

	if successCount == totalCount {
		fmt.Println("🎉 All providers are ready for benchmarking!")
	} else {
		fmt.Println("⚠️  Some providers failed connection test. Check your configuration.")
		return fmt.Errorf("connection test failed for %d provider(s)", totalCount-successCount)
	}

	return nil
}
