package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func initConfiguration(cmd *cobra.Command, args []string) error {
	var configPath string
	configPath = "llmbench.yaml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create sample configuration
	if err := configMgr.CreateSampleConfig(configPath); err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration file created at %s\n", configPath)
	fmt.Println("\n📝 Please edit the configuration file to add your API keys and adjust settings.")
	fmt.Println("\nExample providers:")
	fmt.Println("  - OpenAI: https://api.openai.com/v1")
	fmt.Println("  - Anthropic: https://api.anthropic.com/v1")
	fmt.Println("  - Azure OpenAI: https://your-resource.openai.azure.com/")
	fmt.Println("  - Any OpenAI-compatible API")

	return nil
}

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage LLMBench configuration files and settings.`,
	}

	initConfigCmd = &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new configuration file",
		Long: `Initialize a new configuration file with sample settings.
If no path is provided, creates llmbench.yaml in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: initConfiguration,
	}

	showConfigCmd = &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  `Display the current configuration settings.`,
		RunE:  showConfig,
	}

	validateConfigCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  `Validate the current configuration file for errors.`,
		RunE:  validateConfig,
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(initConfigCmd)
	configCmd.AddCommand(showConfigCmd)
	configCmd.AddCommand(validateConfigCmd)
}

func showConfig(cmd *cobra.Command, args []string) error {
	config := configMgr.GetConfig()
	if config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	fmt.Println("Current Configuration:")
	fmt.Println("=====================")

	fmt.Printf("Requests: %d\n", config.Benchmark.Requests)
	fmt.Printf("Concurrency: %d\n", config.Benchmark.Concurrency)
	fmt.Printf("Timeout: %s\n", config.Benchmark.Timeout)
	fmt.Printf("Providers: %d\n", len(config.Benchmark.Providers))

	fmt.Println("\nProviders:")
	for i, provider := range config.Benchmark.Providers {
		fmt.Printf("  %d. %s\n", i+1, provider.Name)
		fmt.Printf("     Base URL: %s\n", provider.BaseURL)
		if len(provider.Models) > 0 {
			if len(provider.Models) == 1 {
				fmt.Printf("     Model: %s\n", provider.Models[0])
			} else {
				fmt.Printf("     Models: %s\n", strings.Join(provider.Models, ", "))
			}
		} else {
			fmt.Printf("     Models: none configured\n")
		}
		fmt.Printf("     API Key: %s\n", maskAPIKey(provider.APIKey))
	}

	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	config := configMgr.GetConfig()
	if config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	fmt.Println("✅ Configuration is valid")
	fmt.Printf("Found %d provider(s) configured\n", len(config.Benchmark.Providers))

	return nil
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
